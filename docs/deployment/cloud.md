# Cloud Setup Guide

Deploy the **Sequencer** and **Prover** on a Linux cloud server as systemd services.

> [!NOTE]
> This supersedes an earlier version of this guide built around an Indexer-centric
> design (contracts named `TradeRegistry`/`StateManager`, indexer as the primary
> service). The live pipeline today is Sequencer → Prover → `NowaRollup`. The Indexer
> is optional/legacy — see [architecture/overview.md](../architecture/overview.md#indexer-indexer--legacy-optional).
> Its own deployment steps aren't covered here; add an `indexer` service by mirroring
> the Prover unit below with `indexer-bin start` if you need it.

## Prerequisites

- Linux server (Ubuntu 22.04 LTS recommended — see the root `README.md`'s stated minimum: 16GB RAM, 4 cores, 50GB SSD)
- `sudo` access
- Git, Make, curl installed

---

## 1. SSH Key Setup

```bash
ssh-keygen -t ed25519 -C "nowa-zk"
cat ~/.ssh/id_ed25519.pub
```
Add it under GitHub → repo → Settings → Deploy Keys, then test:
```bash
ssh -T git@github.com
```

---

## 2. Install Dependencies

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y make git build-essential curl jq python3

curl -OL https://go.dev/dl/go1.24.10.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.10.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
go version

curl -L https://foundry.paradigm.xyz | bash
source ~/.bashrc
foundryup
forge --version
```

---

## 3. Clone, Set Up Keys, Build

```bash
git clone git@github.com:nowafinance/nowa-zk.git ~/nowa-zk
cd ~/nowa-zk

cp .env.example .env
nano .env
# Set at minimum: L1_RPC_URL, PRIVATE_KEY (funded — pays for deploy + every batch submission)

make setup   # keys → ~/.nowa-zk/keys/, regenerates contracts/src/generated/Verifier.sol
make build   # sequencer-bin, prover-bin, contracts
```

`chmod 600 .env` — it holds a private key.

---

## 4. Deploy Contracts

```bash
set -a
source .env
set +a

make deploy
```
This runs `forge script script/Deploy.s.sol --rpc-url $L1_RPC_URL --broadcast`, deploying
`Verifier.sol` then `NowaRollup.sol`, and copies the resulting addresses to
`~/.nowa-zk/deployments.json` — both the Sequencer (for its deposit watcher) and the
Prover auto-load `NowaRollup`'s address from there.

**Register any ERC20 tokens you want tradeable** (the deploy script doesn't do this —
`registerToken` is owner-only, call it per token after deploy):
```bash
ROLLUP=$(jq -r '.NowaRollup' ~/.nowa-zk/deployments.json)
cast send $ROLLUP "registerToken(address)" <TOKEN_ADDRESS> --rpc-url $L1_RPC_URL --private-key $PRIVATE_KEY
```

> [!IMPORTANT]
> **Bootstrap `stateRoot` — required after every fresh deploy, unconditionally.**
> A fresh `NowaRollup` starts at `stateRoot = 0` — but no real Sequencer tree ever
> roots to `0`, empty or not. A depth-28 SMT's empty root is the MiMC hash of 28 levels
> of zero-nodes
> (`18793058299019184980413965763163005521513826601986169737543200321140307321520`),
> not literal `0`. Skip this and **every `submitBatch()` reverts with "Invalid old
> state root" — spending real gas on each failed attempt.** Sync it once, while
> `batchCount == 0`:
> ```bash
> set -a; source .env; set +a   # cast does NOT auto-load .env like the Go binaries do
>
> ROLLUP=$(jq -r '.NowaRollup' ~/.nowa-zk/deployments.json)
> OLDROOT_DEC=$(curl -s http://localhost:8080/batch/1 | jq -r '.old_root')   # batch #1, not /batch/latest — the Prover always starts there on a fresh checkpoint
> OLDROOT_HEX=$(python3 -c "print('0x' + format(int('$OLDROOT_DEC'), '064x'))")
> cast send $ROLLUP "setStateRoot(bytes32)" $OLDROOT_HEX --rpc-url $L1_RPC_URL --private-key $PRIVATE_KEY
> ```
> Redeploying again doesn't avoid this either — a fresh `NowaRollup` always starts at
> `stateRoot = 0` regardless of deploy count.

---

## 5. Directory Setup

`~/.nowa-zk/` (keys, deployments, per-service data) is created automatically by the
`make` targets above and by the services below. No `/var/lib/nowa-zk` or `/etc/nowa-zk`
convention is needed — everything the binaries read/write lives under `~/.nowa-zk/` for
the user running them.

---

## 6. Systemd Services

Set the username the services will run as:
```bash
USERNAME=nowa
```

### Sequencer Service

```bash
sudo tee /etc/systemd/system/nowa-sequencer.service > /dev/null <<EOF
[Unit]
Description=Nowa-ZK Sequencer
After=network-online.target

[Service]
User=$USERNAME
Group=$USERNAME
WorkingDirectory=/home/$USERNAME/.nowa-zk/sequencer
EnvironmentFile=/home/$USERNAME/nowa-zk/.env
Environment=ROLLUP_CONTRACT_ADDRESS=REPLACE_WITH_NOWAROLLUP_ADDRESS
ExecStart=/home/$USERNAME/nowa-zk/build/sequencer-bin
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```
Replace `REPLACE_WITH_NOWAROLLUP_ADDRESS` with `jq -r '.NowaRollup' ~/.nowa-zk/deployments.json`,
or drop the `Environment=` line and instead put `ROLLUP_CONTRACT_ADDRESS=0x...` directly
in `.env`. `WorkingDirectory` matters — it's what makes the Sequencer's relative LevelDB
path (`./nowa_state_db`) land under `~/.nowa-zk/sequencer/` instead of somewhere random.

### Prover Service

```bash
sudo tee /etc/systemd/system/nowa-prover.service > /dev/null <<EOF
[Unit]
Description=Nowa-ZK Prover
After=network-online.target nowa-sequencer.service

[Service]
User=$USERNAME
Group=$USERNAME
WorkingDirectory=/home/$USERNAME/nowa-zk
EnvironmentFile=/home/$USERNAME/nowa-zk/.env
ExecStart=/home/$USERNAME/nowa-zk/build/prover-bin start --keys-dir /home/$USERNAME/.nowa-zk/keys --indexer-url http://localhost:8080
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```
The Prover auto-loads `NowaRollup`'s address from `~/.nowa-zk/deployments.json` and its
private key / RPC URL from the `EnvironmentFile` (`PRIVATE_KEY`, `L1_RPC_URL`) — no need
to pass `--contract`/`--private-key` explicitly unless you want to override them.

---

## 7. Start Services

```bash
sudo systemctl daemon-reload
sudo systemctl enable nowa-sequencer nowa-prover
sudo systemctl start nowa-sequencer

# confirm it's up before starting the prover
sudo systemctl status nowa-sequencer
curl http://localhost:8080/batch/count

sudo systemctl start nowa-prover
sudo journalctl -u nowa-prover -f
```

---

## 8. Testing This Deployment

Process-up isn't the same as "it actually proves and settles a trade." Confirm each
stage in order — this is the same sequence covered for local dev in
[../testing.md](../testing.md), adapted for a server you reach over SSH.

### 8a. Services are alive

```bash
sudo systemctl status nowa-sequencer nowa-prover
curl http://localhost:8080/orderbook?token_id=1
```
There's no `/status` or `/health` endpoint on the Sequencer today — liveness is "does
`/orderbook` respond."

### 8b. Drive a real matched trade

From the server (or over an SSH tunnel to its `:8080`):
```bash
cd ~/nowa-zk/sequencer
go run ./cmd/cli/test_client.go
curl http://localhost:8080/batch/count       # → {"count":1}
curl http://localhost:8080/batch/latest      # sealed batch with real Merkle paths
```

### 8c. Watch the Prover actually prove and submit

```bash
sudo journalctl -u nowa-prover -f
```
Expect, in order:
```
📦 Processing batch #1
🔐 Generating proof...
🕵️ Verifying proof locally...
📤 Submitting proof + EIP-4844 DA blob to L1...
```

If it stalls silently after "Starting prover loop..." with no "Processing batch #N"
line even though `/batch/count` is non-zero, the Prover's checkpoint store
(`~/.nowa-zk/prover/data`) likely has a stale `last_processed_batch` from a previous
deployment on this same host — `make clean-data` clears it (keys/deployments are
preserved) and restart the service.

If it reaches "Submitting..." and then logs `❌ L1 tx reverted`, see the
`setStateRoot` bootstrap step in §4 above — an unsynced `stateRoot` is the most common
cause, and each retry spends real gas, so fix it before letting the service keep
auto-retrying.

### 8d. Confirm it actually landed on L1

```bash
ROLLUP=$(jq -r '.NowaRollup' ~/.nowa-zk/deployments.json)
cast call $ROLLUP "batchCount()(uint64)" --rpc-url $L1_RPC_URL
cast call $ROLLUP "stateRoot()(bytes32)" --rpc-url $L1_RPC_URL
```
`batchCount` should have incremented and `stateRoot` should equal the batch's
`new_root` (converted to hex) from step 8b. Cross-check the submission transaction on
a block explorer (Sepolia: `https://sepolia.etherscan.io/address/$ROLLUP`) — look for a
successful `submitBatch` call with a blob attached.

---

## 9. Restart After Clearing All Data

Once a batch has been submitted, a contract's `stateRoot` and the Sequencer's tree are
permanently linked — you can't just wipe one side. Full working sequence:

```bash
# 1. Stop services
sudo systemctl stop nowa-sequencer nowa-prover

# 2. Wipe all local state
make clean-sequencer-state
rm -f ~/.nowa-zk/sequencer/test_client.lock
make clean-data

# 3. Fresh contract (a used one's batchCount > 0 blocks re-bootstrapping)
set -a; source .env; set +a
make deploy

# 4. Restart the Sequencer
sudo systemctl start nowa-sequencer

# 5. Generate trades — one Alice/Bob pair placing N trades in one run
cd ~/nowa-zk/sequencer
go run ./cmd/cli/test_client.go --count 100
cd ..

# 6. Bootstrap stateRoot from batch #1 (always #1 — see §4's note above)
ROLLUP=$(jq -r '.NowaRollup' ~/.nowa-zk/deployments.json)
OLDROOT_DEC=$(curl -s http://localhost:8080/batch/1 | jq -r '.old_root')
OLDROOT_HEX=$(python3 -c "print('0x' + format(int('$OLDROOT_DEC'), '064x'))")
cast send $ROLLUP "setStateRoot(bytes32)" $OLDROOT_HEX --rpc-url $L1_RPC_URL --private-key $PRIVATE_KEY

# 7. Start the Prover — processes batch #1, #2, ... in order
sudo systemctl start nowa-prover
sudo journalctl -u nowa-prover -f
```

> [!NOTE]
> Each batch is one real L1 transaction (`BatchSize = 1`) — `--count 100` means 100
> proofs and 100 gas-spending submissions if the Prover runs to completion.

**Partial resets** (skip the full sequence if you don't need a fresh contract):
```bash
make clean-data              # wipe Prover/Indexer DBs only, keep keys & deployments
make clean-sequencer-state   # wipe the Sequencer's Merkle tree only (stop the service first!)
```
For wiping *everything* including keys (not just data), see [cloud-clear.md](./cloud-clear.md).

---

## Manual Verification on a Block Explorer (optional)

```bash
cd ~/nowa-zk/contracts
cat ~/nowa-zk/contracts/deployments/deployments.json   # {"Verifier": "0x...", "NowaRollup": "0x..."}
forge config | grep solc   # confirm the compiler version to enter in the explorer UI
```

```bash
# Verifier
forge verify-contract <VERIFIER_ADDRESS> src/generated/Verifier.sol:Verifier \
  --rpc-url $L1_RPC_URL --etherscan-api-key $ETHERSCAN_API_KEY

# NowaRollup (constructor args: verifier address + initial state root)
forge verify-contract <NOWAROLLUP_ADDRESS> src/NowaRollup.sol:NowaRollup \
  --rpc-url $L1_RPC_URL --etherscan-api-key $ETHERSCAN_API_KEY \
  --constructor-args $(cast abi-encode "constructor(address,bytes32)" <VERIFIER_ADDRESS> <INITIAL_ROOT>)
```
`make verify-contracts` does exactly this for you (Sepolia, chain ID `11155111`) if
`ETHERSCAN_API_KEY` is set in `.env`.

---

## Troubleshooting

See [operations/troubleshooting.md](../operations/troubleshooting.md) for the common
failure modes (stale keys, wrong contract address, `.env` not loading, disk/log
management).
