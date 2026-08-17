# Troubleshooting Guide

Common issues running Nowa-ZK's Sequencer + Prover pipeline.

---

## Environment / Setup Issues

### `.env` not loading when running from a subdirectory

**Symptom:**
```bash
$ cd prover && go run ./cmd/prover start ...
❌ Private key required.
```

**Cause:** the Prover loads `.env` via `godotenv.Load()`, which only checks the
**current working directory**. `.env` lives at the repo root; if you `cd`'d into
`prover/` first (which you must, since it's its own Go module), it silently isn't found.

**Solution:** export it explicitly before running:
```bash
export $(grep -v '^\s*#' ../.env | xargs)
go run ./cmd/prover start --keys-dir ~/.nowa-zk/keys --indexer-url http://localhost:8080
```
Or just use `make run-prover` / `make run-sequencer` from the repo root — the Makefile
targets load `.env` for you.

### `contract source info format must be <path>:<contractname>`

**Solution:** `make deploy` calls forge without a `:Deploy` suffix and that's correct for
this repo (`Deploy.s.sol` has exactly one `Script` contract, so `forge` auto-detects it):
```bash
forge script script/Deploy.s.sol --rpc-url $L1_RPC_URL --private-key $PRIVATE_KEY --broadcast
```
If you're getting this error some other way, check for typos in the script path, not a
missing contract-name suffix.

---

## ZK Proof Generation Issues

### `invalid witness size, got N, expected M`

**Cause:** the compiled circuit backing `--keys-dir` doesn't match
`prover/circuits/state_circuit.go`. Almost always means you pointed at the wrong
directory — **`prover/keys/` in the repo is git-ignored and may hold stale local
artifacts from an older circuit version.**

**Solution:**
```bash
ls -lh ~/.nowa-zk/keys/     # this is the correct directory (what `make setup` writes to)
# should show state.ccs/state.pk/state.vk or rollup.r1cs/rollup.pk/rollup.vk

# if missing or stale, regenerate:
make setup
```

### `constraint #N is not satisfied`

**Two distinct causes — check both:**

1. **Key/circuit mismatch that got past the witness-size check.** Same fix as above:
   `make setup`, and make sure `contracts/src/generated/Verifier.sol` was regenerated
   (and, if already deployed on-chain, that the deployed `NowaRollup` actually points at
   a `Verifier` built from these same keys — see [project-update.md](./project-update.md)).

2. **The batch's `circuit_signature` fields aren't valid for the message the circuit
   actually checks.** The circuit verifies EdDSA over
   `MiMC(OpType, pubX, pubY, baseIndex, quoteIndex)` — **not** the same message the
   Sequencer's order-matching signature check uses. If you hand-built a test order and
   reused the order-intent signature as `circuit_signature`, this is why it fails. See
   `sequencer/cmd/cli/test_client.go` for the correct two-signature construction, or
   [architecture/data-flow.md](../architecture/data-flow.md#1-order-submission).

### `send blob tx: unexpected eip-4844 sidecar after osaka`

**Cause:** the connected L1 node has activated the Osaka hardfork (EIP-7594/PeerDAS),
which rejects the old single-proof (`BlobSidecarVersion0`) blob format outright.

**Solution:** `prover/internal/da/blob.go` must build `BlobSidecarVersion1` sidecars
using `kzg4844.ComputeCellProofs` — confirm you're running a build that includes this
(check the file directly if unsure; `go-ethereum` v1.16.7, already the pinned version in
`prover/go.mod`, supports cell proofs natively — no dependency bump needed). No
transaction is broadcast when this error fires — it's rejected before inclusion, so no
gas is spent on a failed attempt.

### `L1 tx reverted` after "Submitting proof + EIP-4844 DA blob to L1..."

The proof generated and verified fine, and the transaction actually landed on-chain —
it just reverted, which still spends gas.

**Diagnose the actual revert reason** rather than guessing:
```bash
set -a; source .env; set +a   # cast does NOT auto-load .env like the Go binaries do —
                               # without this you'll get "a value is required for
                               # '--rpc-url'" instead of a useful answer

ROLLUP=$(jq -r '.NowaRollup' ~/.nowa-zk/deployments.json)
cast call $ROLLUP "stateRoot()(bytes32)" --rpc-url $L1_RPC_URL
```

**Most likely cause: `"Invalid old state root"`.** `submitBatch()` requires
`_oldRoot == stateRoot`. A freshly-deployed `NowaRollup` starts at `stateRoot = 0` — but
no real Sequencer tree ever roots to `0`, empty or not (a depth-28 SMT's empty root is
the MiMC hash of 28 levels of zero-nodes, not literal `0`). **This means the fix below
is required after every fresh deploy, unconditionally** — not only when the Sequencer
already has accounts. Sync them once (owner-only, only works while `batchCount == 0`):
```bash
OLDROOT_DEC=$(curl -s http://localhost:8080/batch/1 | jq -r '.old_root')   # batch #1, not /batch/latest — the Prover always starts there on a fresh checkpoint
OLDROOT_HEX=$(python3 -c "print('0x' + format(int('$OLDROOT_DEC'), '064x'))")
cast send $ROLLUP "setStateRoot(bytes32)" $OLDROOT_HEX --rpc-url $L1_RPC_URL --private-key $PRIVATE_KEY
```
See [project-update.md](./project-update.md) if you've already submitted at least one
batch on this contract and need to resync after a redeploy instead — `setStateRoot`
won't work once `batchCount > 0`.

> [!NOTE]
> Don't reach for `make clean-sequencer-state` as a shortcut here — it only deletes the
> on-disk LevelDB directory. If the Sequencer process is still running, it already has
> that DB open and keeps serving unchanged (`/batch/latest` will show the exact same
> `old_root` as before). And even after a real restart, the fresh tree still won't root
> to `0` — you'd run the `setStateRoot` fix above anyway. We hit both of these live.

**Other possible causes**, check with `cast call` before assuming it's the root:
- `"Not an authorized prover"` — the address behind `PRIVATE_KEY` isn't `isProver[...] == true` on this contract (only the deployer is auto-authorized; `setProver()` adds others, owner-only).
- `"Empty data hash"` / `"DA blob required"` — shouldn't happen via the normal Prover code path, but would indicate the blob sidecar wasn't actually attached to the transaction.

### `"Invalid old state root"` on batch #2+ (not #1) — even though batch #1 already settled

This is a **different bug** from the bootstrap case above — don't reach for
`setStateRoot()` here, it won't help (see why below).

**Symptom:** batch #1 proves and settles cleanly. Batch #2 (or later) reverts with
`Invalid old state root`, and `cast call $ROLLUP "stateRoot()(bytes32)"` shows the
on-chain root *does* equal the previous batch's `new_root` — but the new batch's
`old_root` doesn't match it anyway.

**Cause:** `GET /account` (`sequencer/internal/api/account.go`'s `openBalance`)
lab-credits a brand-new pubkey by writing straight to the Merkle tree —
`tree.SetBalance(acc)` — with **no corresponding `StateTransition` recorded**. That
mutation is completely invisible to the batcher/prover. If *any* new pubkey gets
onboarded (via `/account`, `/balance`, or a first-time trader inside `applyTrade`)
between two batches you intend to submit, the later batch's `old_root` silently
diverges from the earlier batch's `new_root`, and `submitBatch()` will reject it
forever — there is no fix for that specific already-sealed batch.

**Confirm it's this** by diffing the two roots directly:
```bash
curl -s http://localhost:8080/batch/1 | jq -r '.new_root'
curl -s http://localhost:8080/batch/2 | jq -r '.old_root'
# if these differ, this is the bug
```

**Why `setStateRoot()` can't bail you out here:** it only works `while batchCount == 0`
— and by the time you've hit this, batch #1 (or later) has already succeeded, so
`batchCount` is already `≥ 1`. The broken batch is permanently stuck; there is no
recovery path for it.

**Fix — prevention, not cure:**
- `sequencer/cmd/cli/test_client.go` now takes `--count N` to place `N` matched trades
  in a *single* run, registering its one Alice/Bob pair exactly once up front — every
  batch sealed within that run chains correctly. See
  [testing.md](../testing.md#stress-testing-with-many-trades---count).
- The tool also refuses to run a second time on its own (writes
  `~/.nowa-zk/sequencer/test_client.lock`), specifically to stop this from recurring.
- If you're already stuck: the only way forward is `make deploy` (fresh contract,
  `batchCount` back to `0`) + the `setStateRoot` bootstrap from the section above,
  synced to whatever the Sequencer's *current* `old_root` actually is right now.

### Prover silently does nothing (no "Processing batch #N" logged, no errors)

**Symptom:** the Prover's been running for a while, `/batch/count` shows a real sealed
batch waiting, but the log never shows `📦 Processing batch #N` — it just sits after
`🚀 Starting prover loop...`.

**Cause:** the Prover's checkpoint store (`--data-dir`, default
`~/.nowa-zk/prover/data`) already has a `last_processed_batch` value at or past the
latest sealed batch — usually left over from a previous run on the same machine
(including old testing sessions from before you were working on this deployment; the
checkpoint doesn't know the Sequencer's batch history was reset). The Prover computes
`nextBatchNum = last_processed_batch + 1`; if that's already `> latestBatch.BatchID`,
it decides there's nothing new and just sleeps — this path doesn't log anything, which
is itself worth treating as a known rough edge.

**Solution:**
```bash
sudo systemctl stop nowa-prover   # or Ctrl+C
make clean-data                  # wipes the checkpoint only — keys/deployments untouched
sudo systemctl start nowa-prover
```

### Prover using the wrong contract address

**Symptom:**
```
📄 Loaded NowaRollup from /home/user/.nowa-zk/deployments.json
❌ Failed to generate proof: constraint #N is not satisfied
```
after a fresh `make deploy`.

**Cause:** `~/.nowa-zk/deployments.json` still has the old address.

**Solution:**
```bash
sudo systemctl stop nowa-prover   # or Ctrl+C if running manually

cp ~/nowa-zk/contracts/deployments/deployments.json ~/.nowa-zk/deployments.json
cat ~/.nowa-zk/deployments.json   # confirm NowaRollup/Verifier are the NEW addresses

# start fresh against the new contract
make clean-data   # wipes ~/.nowa-zk/prover/data (checkpoint only, not keys)
sudo systemctl start nowa-prover
```

### Prover halted / refuses to restart

```bash
./build/prover-bin start --keys-dir ~/.nowa-zk/keys --clear-halt
```
Paranoid mode halts the Prover after repeated local-verification failures rather than
silently submitting a possibly-bad proof. Clearing the halt resumes from the last
checkpoint — only do this once you've actually fixed the underlying cause (see the
constraint/witness errors above), not as a reflex.

---

## Service Not Starting

### Sequencer won't start

```bash
sudo journalctl -u nowa-sequencer -n 50
```
- **Port in use**: `sudo lsof -i :8080` → `kill -9 <PID>`.
- **LevelDB lock**: `resource temporarily unavailable` on startup means another process
  already has `~/.nowa-zk/sequencer/nowa_state_db` open — find and kill it, don't delete
  the DB unless you actually want to lose state.
- **Permission issues**: `sudo chown -R $USER:$USER ~/.nowa-zk`.

### Prover won't start

```bash
sudo journalctl -u nowa-prover -n 50
```
- **Keys not found**: `ls -lh ~/.nowa-zk/keys/` — regenerate with `make setup` if empty.
- **Wrong/missing contract address**: `cat ~/.nowa-zk/deployments.json`; confirm it's
  live: `cast code <ADDRESS> --rpc-url $L1_RPC_URL` (should return non-empty bytecode).
- **Insufficient funds**: `cast balance <PROVER_ADDRESS> --rpc-url $L1_RPC_URL` — the
  Prover's key pays gas for every `submitBatch()`.

---

## Build Failures

```bash
cd sequencer && go clean -cache && go mod tidy && cd ..
cd prover && go clean -cache && go mod tidy && cd ..
make build

# Forge
cd contracts && forge clean && forge build
```

---

## Getting Help

```bash
sudo journalctl -u nowa-sequencer -f
sudo journalctl -u nowa-prover -f
sudo systemctl status nowa-sequencer nowa-prover
htop
df -h
```
