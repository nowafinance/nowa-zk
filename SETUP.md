# Setup Guide

This document covers two scenarios:

1. **[Local Setup](#1-local-setup)** — run everything on your own machine (Anvil
   local chain, single terminal set) for development.
2. **[VM / Server Setup](#2-vm--server-setup)** — run the same components as
   long-lived background services on a cloud VM (Sepolia or another real L1 RPC,
   not Anvil).

Both use the same `Makefile` targets — the VM section just wraps them in `systemd`
units instead of foreground terminals.

---

## Prerequisites

- **OS**: Ubuntu 22.04 LTS (or compatible Linux)
- **Hardware**: 16GB+ RAM, 4+ CPU cores, 50GB+ SSD — the Prover (Groth16 proving) is
  compute- and memory-intensive
- **Go**: 1.25.7+ (sequencer, prover), 1.24+ (indexer)
- **Foundry**: latest (`forge`, `cast`, `anvil`)
- **Make**, **Git**, **jq**, **python3**

Install Foundry if you don't have it, and make sure it's on `PATH` (it installs to
`~/.foundry/bin`, which is not always on `PATH` by default):

```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
export PATH="$HOME/.foundry/bin:$PATH"   # add to ~/.bashrc to persist
```

Install Go 1.25.7+ from https://go.dev/dl/ if not already present, and confirm:

```bash
go version
```

---

## 1. Local Setup

### 1.1 Clone & configure

```bash
git clone https://github.com/nowafinance/nowa-zk.git
cd nowa-zk
cp .env.example .env
```

Edit `.env`:

```bash
L2_RPC_URL=http://0.0.0.0:8545
L1_RPC_URL=http://127.0.0.1:8545       # Anvil, for local dev
PRIVATE_KEY=0xYOUR_DEPLOYER_KEY         # Anvil prints funded test keys on startup
TRAFFIC_GEN_KEY=0xYOUR_TRAFFIC_GEN_KEY
INDEX_FROM_BLOCK=0
PROVER_API=http://0.0.0.0:8081
TARGET_CONTRACT=              # filled in after deploy, or leave blank
ETHERSCAN_API_KEY=            # only needed for verify-contracts on a real network
```

### 1.2 One-time build & key generation

```bash
make clean-artifacts      # clear stale build output (safe on a fresh clone)
make setup                # generates Groth16 proving/verifying keys + Verifier.sol
make build                # builds sequencer, prover, indexer, contracts
make test                 # run the full test suite
```

`make setup` writes proving/verifying keys to `~/.nowa-zk/keys` and generates
`contracts/src/generated/Verifier.sol` — this only needs to run once per machine
(re-run only if you change the circuit).

### 1.3 Run it — 4 terminals

**Terminal 1 — local chain:**

```bash
export PATH="$HOME/.foundry/bin:$PATH"
make anvil          # starts Anvil on :8545
```

**Terminal 2 — deploy contracts:**

```bash
export PATH="$HOME/.foundry/bin:$PATH"
make deploy          # deploys Verifier + NowaRollup to $L1_RPC_URL
```

`make deploy` auto-computes `INITIAL_STATE_ROOT` from the Sequencer's current local
tree (no manual bootstrap step needed for a fresh deploy). Deployed addresses are
written to `~/.nowa-zk/deployments.json`.

If you also need a local test ERC-20 to deposit/trade with:

```bash
cd contracts && forge script script/DeployMockToken.s.sol --rpc-url $L1_RPC_URL --broadcast
```

**Terminal 3 — sequencer:**

```bash
make run-sequencer    # starts on :8080, reads NowaRollup addr from deployments.json
```

**Terminal 4 — prover:**

```bash
export PATH="$HOME/.foundry/bin:$PATH"
make run-prover       # polls the sequencer, builds Groth16 proofs, submits batches to L1
```

Optional — legacy L2 indexer (only needed if something consumes its API):

```bash
make run-indexer
```

Check batch progress at any time:

```bash
make check-batch
```

### 1.4 Reset local state

```bash
make clean-data       # clears indexer/prover databases
make clean-global      # clears ~/.nowa-zk/ (keys, deployments, sequencer state)
```

Run these before re-deploying from scratch, or if the Sequencer/Prover state gets
into a bad spot during development.

---

## 2. VM / Server Setup

Same components, same `Makefile` targets, pointed at a real L1 RPC (e.g. Sepolia)
instead of Anvil, running as `systemd` services instead of foreground terminals.
No cloud-provider-specific tooling is required — this works on any Ubuntu 22.04 VM
(GCP, AWS, bare metal, etc.).

### 2.1 Provision the box

```bash
sudo apt update && sudo apt install -y make git jq python3 build-essential

# Go
curl -LO https://go.dev/dl/go1.25.7.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.25.7.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile.d/go.sh
source /etc/profile.d/go.sh

# Foundry
curl -L https://foundry.paradigm.xyz | bash
~/.foundry/bin/foundryup
echo 'export PATH=$PATH:'"$HOME"'/.foundry/bin' >> ~/.bashrc
export PATH="$HOME/.foundry/bin:$PATH"
```

Create a dedicated non-root user to run the services (recommended):

```bash
sudo useradd -m -s /bin/bash nowa
sudo su - nowa
```

### 2.2 Clone, configure, build

```bash
git clone https://github.com/nowafinance/nowa-zk.git
cd nowa-zk
cp .env.example .env
```

Edit `.env` for a real network:

```bash
L2_RPC_URL=http://127.0.0.1:8545
L1_RPC_URL=https://ethereum-sepolia-rpc.publicnode.com   # or your own RPC provider
PRIVATE_KEY=0xYOUR_DEPLOYER_KEY            # funded on Sepolia, keep this secret
TRAFFIC_GEN_KEY=0xYOUR_TRAFFIC_GEN_KEY
INDEX_FROM_BLOCK=0
PROVER_API=http://0.0.0.0:8081
ETHERSCAN_API_KEY=YOUR_ETHERSCAN_API_KEY_HERE
```

`.env` holds live private keys — make sure it's not world-readable
(`chmod 600 .env`) and never commit it (it's already git-ignored).

```bash
export PATH="$HOME/.foundry/bin:$PATH"
make clean-artifacts
make setup
make build
make test
```

### 2.3 Deploy once

```bash
export PATH="$HOME/.foundry/bin:$PATH"
make deploy
make verify-contracts   # optional — verifies on Etherscan, needs ETHERSCAN_API_KEY
```

Note the deployed `NowaRollup` address from `~/.nowa-zk/deployments.json` — the
services below read it automatically from that file, but it's worth recording
separately for `reconstruct-proof`/`claim-escape` use later.

### 2.4 Run as systemd services

The repo doesn't ship `.service` units — these are minimal wrappers around the same
`make run-*` targets used locally, so the deployment behavior matches exactly what
was tested. Adjust `User=`/`WorkingDirectory=`/`PATH` to your setup.

**`/etc/systemd/system/nowa-sequencer.service`**

```ini
[Unit]
Description=Nowa-ZK Sequencer
After=network.target

[Service]
Type=simple
User=nowa
WorkingDirectory=/home/nowa/nowa-zk
Environment=PATH=/usr/local/go/bin:/home/nowa/.foundry/bin:/usr/bin:/bin
ExecStart=/usr/bin/make run-sequencer
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

**`/etc/systemd/system/nowa-prover.service`**

```ini
[Unit]
Description=Nowa-ZK Prover
After=network.target nowa-sequencer.service
Requires=nowa-sequencer.service

[Service]
Type=simple
User=nowa
WorkingDirectory=/home/nowa/nowa-zk
Environment=PATH=/usr/local/go/bin:/home/nowa/.foundry/bin:/usr/bin:/bin
ExecStart=/usr/bin/make run-prover
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Optional — legacy indexer, same pattern:

**`/etc/systemd/system/nowa-indexer.service`**

```ini
[Unit]
Description=Nowa-ZK Indexer
After=network.target

[Service]
Type=simple
User=nowa
WorkingDirectory=/home/nowa/nowa-zk
Environment=PATH=/usr/local/go/bin:/home/nowa/.foundry/bin:/usr/bin:/bin
ExecStart=/usr/bin/make run-indexer
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now nowa-sequencer
sudo systemctl enable --now nowa-prover
# sudo systemctl enable --now nowa-indexer   # only if you need it

# Logs
journalctl -u nowa-sequencer -f
journalctl -u nowa-prover -f
```

The Sequencer listens on `:8080`, the Prover polls it there by default (per
`run-prover`'s `--indexer-url http://localhost:8080`) — keep both on the same host,
or open the Sequencer's port to the Prover's host if you split them across VMs.

### 2.5 Firewall

Only the Sequencer's `:8080` API needs to be reachable by clients (order
submission, `GET /proof` for the escape hatch). The Prover talks to the Sequencer
over `localhost` and to L1 over outbound HTTPS — it does not need an inbound port
open. Don't expose `:8080` more broadly than necessary; put it behind a reverse
proxy/TLS terminator if it needs to be public.

```bash
sudo ufw allow 8080/tcp    # only if the Sequencer API must be reached externally
sudo ufw enable
```

---

## Reference

- `make help` — full list of targets in intended execution order.
- Full target list: `clean-artifacts`, `clean-data`, `clean-global`, `setup`,
  `build`, `test`, `anvil`, `deploy`, `verify-contracts`, `run-sequencer`,
  `run-indexer`, `run-prover`, `check-batch`.
- Escape hatch (if the Sequencer or Prover goes down): see `sequencer/cmd/claim-escape`
  and `sequencer/cmd/reconstruct-proof` — usable independently of the services above.
