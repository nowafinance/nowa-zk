# Local Development Setup

Run the full Nowa-ZK stack — Sequencer, Prover, and contracts — on one machine.

## Prerequisites

- **Go** 1.24.10+, **Foundry** (Forge, Cast, Anvil)
- `make`, `git`, `build-essential`, `curl`, `jq`, `python3`

```bash
sudo apt update && sudo apt install -y make git build-essential curl jq python3

curl -OL https://go.dev/dl/go1.24.10.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.10.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

curl -L https://foundry.paradigm.xyz | bash
source ~/.bashrc
foundryup
```

---

## 1. Clone & Configure

```bash
git clone https://github.com/nowafinance/nowa-zk.git
cd nowa-zk
cp .env.example .env
```

Edit `.env` for a local Anvil run:
```bash
L1_RPC_URL=http://localhost:8545
PRIVATE_KEY=0x...   # a funded Anvil default account works fine
```

> [!NOTE]
> `.env` is only auto-loaded from the repo root. Running a binary directly from
> `sequencer/` or `prover/` needs `export $(grep -v '^#' ../.env | xargs)` first —
> see [Troubleshooting](../operations/troubleshooting.md#env-not-loading-when-running-from-a-subdirectory).

## 2. Generate Keys & Build

```bash
make setup   # circuit compile + trusted setup → ~/.nowa-zk/keys/, generates Verifier.sol
make build   # sequencer-bin, prover-bin, indexer-bin, contracts
```

## 3. Run Tests (optional)

```bash
make test
```
See [../testing.md](../testing.md) for per-component detail.

## 4. First-Time Run

**4 terminals:**

### Terminal 1: Local chain
```bash
make anvil
```
Skip this and set `L1_RPC_URL` to a real testnet instead if you'd rather not run one locally.

### Terminal 2: Deploy contracts
```bash
make deploy
```
Deploys `Verifier.sol` then `NowaRollup.sol`, saves addresses to
`~/.nowa-zk/deployments.json` (Sequencer and Prover both auto-load from there).

> [!IMPORTANT]
> **Always bootstrap `stateRoot` after deploying, before submitting any batch.** A
> fresh `NowaRollup` starts at `stateRoot = 0`, but no real Sequencer tree — empty or
> not — ever actually roots to `0`. Skipping this makes every `submitBatch()` revert
> with `Invalid old state root`, spending real gas each time. Fix (owner-only, works
> only before the first successful submission):
> ```bash
> set -a; source .env; set +a   # cast needs this — it doesn't auto-load .env
> ROLLUP=$(jq -r '.NowaRollup' ~/.nowa-zk/deployments.json)
> OLDROOT_DEC=$(curl -s http://localhost:8080/batch/1 | jq -r '.old_root')   # batch #1, not /batch/latest — the Prover always starts there on a fresh checkpoint
> OLDROOT_HEX=$(python3 -c "print('0x' + format(int('$OLDROOT_DEC'), '064x'))")
> cast send $ROLLUP "setStateRoot(bytes32)" $OLDROOT_HEX --rpc-url $L1_RPC_URL --private-key $PRIVATE_KEY
> ```
> `make clean-sequencer-state` is **not** a substitute — it only deletes the on-disk DB
> and does nothing while the Sequencer is still running, and even a truly empty tree
> doesn't root to `0` anyway. Full details: [Troubleshooting](../operations/troubleshooting.md).

### Terminal 3: Sequencer
```bash
make run-sequencer
```
Matching engine + REST API on `:8080`. State: `~/.nowa-zk/sequencer/nowa_state_db`.

Place a real matched trade:
```bash
make test-sequencer-live
curl http://localhost:8080/batch/latest
```

> [!IMPORTANT]
> **Run this only once per Sequencer/contract lifetime.** It mints a fresh random
> Alice/Bob keypair each time, and onboarding a new account silently advances the tree
> root outside of any batch — a second run breaks the chain between batches and the
> new one can never be submitted. To generate many trades safely instead, use one run
> with `--count`:
> ```bash
> cd sequencer && go run ./cmd/cli/test_client.go --count 1000
> ```
> This registers one Alice/Bob pair once, then places all 1000 trades in that same run,
> so every batch chains correctly. It also locks itself after a successful run
> (`~/.nowa-zk/sequencer/test_client.lock`) and refuses to run again — delete the lock
> or pass `--force` if you really need to. Details: [Troubleshooting](../operations/troubleshooting.md).

### Terminal 4: Prover
```bash
make run-prover
```
Polls the Sequencer, proves each batch, and submits to L1 — that last step spends real
gas.

---

## Optional: Indexer (legacy)

Not required above. Only if working on the Cosmos-indexer subsystem — see
[architecture/overview.md](../architecture/overview.md#indexer-indexer--legacy-optional).
```bash
make run-indexer
```

---

## 5. Restart After Clearing All Data

Once a batch has been submitted, a contract's `stateRoot` and the Sequencer's tree are
permanently linked (see [Troubleshooting](../operations/troubleshooting.md)) — you can't
just wipe one side. This is the full, working sequence to actually start clean:

```bash
# 1. Stop everything: Ctrl+C in the `make run-sequencer` and `make run-prover` terminals

# 2. From the repo root — wipe all local state
make clean-sequencer-state
rm -f ~/.nowa-zk/sequencer/test_client.lock
make clean-data

# 3. Fresh contract (a used one's batchCount > 0 blocks re-bootstrapping)
set -a; source .env; set +a
make deploy

# 4. Restart the Sequencer (separate terminal)
make run-sequencer

# 5. Generate trades — adjust --count, this is one Alice/Bob pair placing N trades
cd sequencer && go run ./cmd/cli/test_client.go --count 100 && cd ..

# 6. Bootstrap stateRoot from batch #1 (always #1 — same reasoning as the Terminal 2 note in §4)
ROLLUP=$(jq -r '.NowaRollup' ~/.nowa-zk/deployments.json)
OLDROOT_DEC=$(curl -s http://localhost:8080/batch/1 | jq -r '.old_root')
OLDROOT_HEX=$(python3 -c "print('0x' + format(int('$OLDROOT_DEC'), '064x'))")
cast send $ROLLUP "setStateRoot(bytes32)" $OLDROOT_HEX --rpc-url $L1_RPC_URL --private-key $PRIVATE_KEY

# 7. Prover (separate terminal) — processes batch #1, #2, ... in order
make run-prover
```

> [!NOTE]
> Each batch is one real L1 transaction (`BatchSize = 1`) — `--count 100` means 100
> proofs and 100 gas-spending submissions if you let the Prover run to completion.

**Partial resets** (skip the full sequence above if you don't need a fresh contract):
```bash
make clean-data              # wipe Prover/Indexer DBs only, keep keys & deployments
make clean-sequencer-state   # wipe the Sequencer's Merkle tree only (stop it first!)
make clean-global            # wipe everything under ~/.nowa-zk/, including keys
```
