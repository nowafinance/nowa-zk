# Testing Guide

Quick reference for testing every component of the Nowa-ZK system, plus a real
end-to-end run.

> [!NOTE]
> This guide assumes **local development** — everything running on one machine across
> a few terminals, per [deployment/local.md](deployment/local.md). Testing an already
> -deployed **cloud** instance (systemd services, SSH, a live Sepolia contract) has its
> own walkthrough: [deployment/cloud.md §8](deployment/cloud.md#8-testing-this-deployment).
> The underlying test commands are the same either way — the cloud guide just adapts
> them for journalctl/systemd instead of foreground terminals.

---

## Quick Setup & Validation

```bash
make clean-artifacts
make setup    # generates ~/.nowa-zk/keys/ + contracts/src/generated/Verifier.sol
make build
make test     # runs contracts + indexer + prover + sequencer unit tests
```

---

## 1. Contract Testing

```bash
cd contracts
forge test -vv

# Specific test
forge test --match-test testDeposit -vvv

# Gas report
forge test --gas-report
```

Deploy locally against Anvil:
```bash
make anvil      # Terminal 1
make deploy     # Terminal 2 — needs L1_RPC_URL=http://localhost:8545 in .env

cat ~/.nowa-zk/deployments.json   # {"NowaRollup": "0x...", "Verifier": "0x..."}
```

**Common error**: `Verifier.sol not found` → run `make setup` first (it generates the
verifier from the compiled circuit before contracts can build against it).

---

## 2. Sequencer Testing

### Unit tests
```bash
cd sequencer
go test ./... -v
# or: make test-sequencer
```
Covers matching (`internal/engine`), the LevelDB SMT (`internal/state`), batching
(`internal/batcher`), and full trade application (`cmd/sequencer/apply_trade_test.go`).

### Live integration test
```bash
# Terminal 1
make run-sequencer
# → "Sequencer API listening on :8080..."

# Terminal 2
make test-sequencer-live
# runs sequencer/cmd/cli/test_client.go: generates one Alice & Bob EdDSA keypair pair,
# registers both accounts (GET /account) ONCE, signs a real order-intent signature AND
# the correct in-circuit authorization signature, then posts a matching sell/buy pair.
```

Confirm the match landed:
```bash
curl http://localhost:8080/orderbook?token_id=1
curl http://localhost:8080/batch/count       # → {"count":1}
curl http://localhost:8080/batch/latest      # full sealed batch, with real Merkle paths
```

**Common error**: `circuit_signature required` — you're hitting `/order` with a payload
that only has the order-intent `signature` field. The circuit-level signature is a
*separate* field (`circuit_signature`) over a different message
(`MiMC(OpType, pubX, pubY, baseIndex, quoteIndex)`) — see
`sequencer/cmd/cli/test_client.go` for how to construct it, or
[architecture/data-flow.md](architecture/data-flow.md#1-order-submission).

### Stress-testing with many trades (`--count`)

```bash
cd sequencer
go run ./cmd/cli/test_client.go --count 1000
curl http://localhost:8080/batch/count   # should have grown by exactly 1000
```
Registers one Alice/Bob pair once, then places `N` matched trades in the same run —
every resulting batch chains correctly (`batch[i].new_root == batch[i+1].old_root`),
which is what actually lets you verify multi-batching / the Prover working through a
backlog, rather than one isolated trade.

> [!CAUTION]
> **Do not run `test_client.go` (or `make test-sequencer-live`) more than once per
> Sequencer/contract lifetime.** `GET /account` lab-credits a new pubkey by writing
> directly to the Merkle tree with no `StateTransition` recorded — invisible to the
> batcher. A second run mints a *new* random keypair, silently advancing the tree root
> between already-sealed batches, and the next batch becomes permanently unsubmittable
> (`setStateRoot()`'s escape hatch only works while `batchCount == 0`). We hit this
> live: batch #1 settled, a second run's batch #2 never could. The tool now enforces
> this itself — it writes `~/.nowa-zk/sequencer/test_client.lock` after a successful
> run and refuses to run again (`--force` to override if you understand the risk; see
> [operations/troubleshooting.md](operations/troubleshooting.md) for full recovery
> options if you're already stuck).

---

## 3. Prover Testing

### Unit tests
```bash
cd prover
go test ./... -v

# Circuit correctness + benchmarks (includes a real Groth16 prove/verify cycle, ~14s)
cd circuits && go test -v -bench=. -benchtime=1x
```

### Setup validation
```bash
ls -lh ~/.nowa-zk/keys/
# Should show: state.ccs (or rollup.r1cs), state.pk (or rollup.pk), state.vk (or rollup.vk)

ls -lh contracts/src/generated/Verifier.sol
```

### Live integration test

Needs a sealed batch from the Sequencer test above, plus a funded `PRIVATE_KEY` and
`L1_RPC_URL` in the root `.env` if you want to go all the way to L1 submission.

```bash
# From the repo root (so .env loads automatically):
make run-prover
```
Or manually, from inside `prover/` (note: `.env` won't auto-load here — see
[Troubleshooting](operations/troubleshooting.md#env-not-loading-when-running-from-a-subdirectory)):
```bash
cd prover
export $(grep -v '^\s*#' ../.env | xargs)
go run ./cmd/prover start --keys-dir ~/.nowa-zk/keys --indexer-url http://localhost:8080
```

Expected sequence:
```
📦 Processing batch #1
🔐 Generating proof...          ← Groth16 prove, ~1-2s for a single-fill batch
🕵️ Verifying proof locally...   ← should always pass if the previous step succeeded
📤 Submitting proof + EIP-4844 DA blob to L1...   ← spends real gas from PRIVATE_KEY
```

> [!WARNING]
> The last step broadcasts a real transaction. Only run it against a network/key you
> intend to spend testnet (or real) gas on.

### Error handling tests (paranoid mode)
```bash
./build/prover-bin start --keys-dir ~/.nowa-zk/keys --test-failure
# Expected: retries → rebuild → halt

./build/prover-bin start --keys-dir ~/.nowa-zk/keys --clear-halt   # resume after halt
./build/prover-bin start --keys-dir ~/.nowa-zk/keys --test-failure --paranoid-mode=false
# Expected: retries only, no rebuild, moves to next batch
```

**Common errors:**
```bash
# "invalid witness size, got N, expected M"
# The compiled keys in --keys-dir don't match the current circuit.
# Fix: make setup   (regenerates keys to match prover/circuits/state_circuit.go)
# Do NOT point --keys-dir at prover/keys/ — that's git-ignored and may be stale.

# "constraint #N is not satisfied"
# Either a genuine key/circuit mismatch (same fix as above), or the batch's
# circuit_signature fields don't authorize the exact message the circuit expects
# (see Sequencer testing above) — check the signing recipe, not just the keys.

# "send blob tx: unexpected eip-4844 sidecar after osaka"
# go-ethereum built the old single-proof (Version0) blob sidecar; the connected
# chain has moved past the Osaka hardfork and rejects that format. Fixed in
# prover/internal/da/blob.go (BlobSidecarVersion1 / cell proofs) — make sure
# you're on a build that includes that fix.
```

---

## 4. Contract Unit Tests Directly Against the Circuit

`contracts/test/NowaRollup.t.sol` exercises `deposit`/`submitBatch`/proof verification
against a `TestVerifier.sol` stub and a canned proof (`contracts/test/data/test_proof.json`).
Regenerate the test proof if the circuit changes — a stale fixture here will fail in a
way that looks like a contract bug but isn't.

---

## 5. Full End-to-End Test

```bash
# Terminal 1: local chain (skip if using a real testnet RPC)
make anvil

# Terminal 2: deploy contracts
make deploy
cat ~/.nowa-zk/deployments.json

# Terminal 3: Sequencer
make run-sequencer

# Terminal 4: drive real trades
make test-sequencer-live
curl http://localhost:8080/batch/latest

# Terminal 5: Prover — proves + submits to L1
make run-prover
```

Watch the Sequencer's stdout for `Matched Trade: ...` / `Sealed batches available: N`,
and the Prover's for the proof-generation → verification → submission sequence above.

---

## 6. Indexer Testing (legacy, optional)

The Indexer (`indexer/`) isn't on the live trading path (see
[architecture/overview.md](architecture/overview.md#indexer-indexer--legacy-optional)),
but it still has its own test suite if you're working on it:
```bash
cd indexer
go test ./... -v
./test.sh   # shell-driven integration smoke, see the script for exact endpoints
```

---

## Quick Reference: Test Commands

| Component | Command | Purpose |
|-----------|---------|---------|
| Everything | `make test` | All unit tests (contracts + indexer + prover + sequencer) |
| Contracts | `cd contracts && forge test` | Contract unit tests |
| Sequencer | `cd sequencer && go test ./...` (or `make test-sequencer`) | Matching, SMT, batching |
| Sequencer (live) | `make test-sequencer-live` (needs `make run-sequencer` running) | Real matched trade against a live REST API |
| Prover | `cd prover && go test ./...` | Circuit, DA blob packing, witness building |
| Circuit benchmarks | `cd prover/circuits && go test -bench=. -benchtime=1x` | Real Groth16 prove/verify timing |
| E2E | See §5 above | Full order → match → proof → L1 |

---

## Test Checklist Before Deploying Anywhere Real

- [ ] `make clean-artifacts && make setup && make build && make test` passes
- [ ] `~/.nowa-zk/keys/` and `contracts/src/generated/Verifier.sol` are freshly
      generated together (never mix keys from one circuit version with a verifier from
      another)
- [ ] Contracts deploy successfully and `~/.nowa-zk/deployments.json` reflects the new
      addresses
- [ ] `make test-sequencer-live` against `make run-sequencer` produces a sealed batch
- [ ] `make run-prover` gets through proof generation + local verification (L1
      submission only if you intend to spend gas)
- [ ] Error handling tested (`--test-failure`, `--clear-halt`)
- [ ] You are aware `NowaRollup.withdraw()` is a placeholder, not an escape hatch — see
      [architecture/overview.md](architecture/overview.md#known-gaps-as-of-2026-08-17)
      before treating any deployment as holding real user funds
