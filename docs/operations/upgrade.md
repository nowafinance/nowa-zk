# Cloud Upgrade Guide

This guide describes how to upgrade the **Nowa-ZK Indexer and Prover** running on a cloud server with systemd services.

> [!IMPORTANT]
> This upgrade process **preserves all chain data**. The indexer state and prover data will remain intact.

---

## Prerequisites

*   SSH access to the server
*   `sudo` privileges
*   Services running via systemd

---

## Upgrade Steps

### Step 1: Stop Services

Stop the running systemd services to ensure no data corruption during the update.

```bash
sudo systemctl stop nowa-zk-indexer nowa-zk-prover
```

Verify they are stopped:

```bash
sudo systemctl status nowa-zk-indexer nowa-zk-prover
```

---

### Step 2: Get Latest Code

Navigate to the project directory and pull the latest changes.

```bash
cd ~/nowa-zk
git pull origin main
```

---

### Step 3: Rebuild Components

#### If Circuit/Keys Changed (Check Release Notes First!)

> [!CAUTION]
> **If the ZK circuit changed, you MUST:**
> 1. Regenerate prover keys
> 2. Regenerate the `RollupVerifier.sol` contract
> 3. **Redeploy ALL contracts** to the chain
> 4. Update the prover service with the new contract address
> 
> The verification keys are cryptographically tied to the circuit. The old deployed contract cannot verify proofs from new keys.

**If circuit changed, follow these steps:**

```bash
cd ~/nowa-zk

# 1. Rebuild prover binary
cd prover
go build -o ../build/prover-bin ./cmd/prover
cd ..

# 2. Regenerate keys AND new RollupVerifier.sol
./build/prover-bin setup --output-dir ./keys --contract-output ./contracts/src/generated

# 3. Rebuild contracts (includes new verifier)
cd contracts
forge build
cd ..

# 4. Redeploy contracts
cd contracts
set -a
source /etc/nowa-zk/.env
set +a
forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast
cd ..
# ⚠️ SAVE THE NEW CONTRACT ADDRESS from deploy output

# 5. Copy new keys to persistent storage
sudo cp -r ./keys/* /var/lib/nowa-zk/prover/keys/
sudo chown -R $USER:$USER /var/lib/nowa-zk

# 6. Update prover systemd service with NEW contract address
sudo nano /etc/systemd/system/nowa-zk-prover.service
# Update: ExecStart=.../prover-bin start --keys-dir ... --contract <NEW_ADDRESS> --private-key ...

# 7. Reload systemd config
sudo systemctl daemon-reload
```

> [!WARNING]
> After redeploying contracts, all previous on-chain state will be lost. Only do this when absolutely necessary.

#### Rebuild Contracts (if needed)

> [!NOTE]
> Only rebuild if contracts changed. Check release notes.

```bash
cd ~/nowa-zk/contracts

# If contracts changed:
# forge build

cd ..
```

#### Rebuild Indexer

```bash
cd ~/nowa-zk/indexer

# Rebuild indexer binary
go build -o ../build/indexer-bin ./cmd/indexer

cd ..
```

#### Rebuild Prover

```bash
cd ~/nowa-zk/prover

# Rebuild prover binary
go build -o ../build/prover-bin ./cmd/prover

cd ..
```

---

### Step 4: Restart Services

Start the services again. They will pick up the newly built binaries automatically.

```bash
sudo systemctl start nowa-zk-indexer nowa-zk-prover
```

---

### Step 5: Verify Upgrade

Check the logs to ensure everything started correctly and is processing blocks.

```bash
# Check Indexer Logs
sudo journalctl -u nowa-zk-indexer -f

# Check Prover Logs (in another terminal)
sudo journalctl -u nowa-zk-prover -f
```

**Look for:**
- ✅ "Indexer started" or similar startup message
- ✅ Block processing continuing from last height
- ✅ No error messages
- ✅ Prover connecting and processing batches

---

## Quick Reference

```bash
# Full upgrade workflow
sudo systemctl stop nowa-zk-indexer nowa-zk-prover
cd ~/nowa-zk
git pull origin main

# Rebuild binaries
cd indexer && go build -o ../build/indexer-bin ./cmd/indexer && cd ..
cd prover && go build -o ../build/prover-bin ./cmd/prover && cd ..

# Restart services
sudo systemctl start nowa-zk-indexer nowa-zk-prover

# Monitor
sudo journalctl -u nowa-zk-indexer -f
```

---

## Rollback

If the upgrade fails, you can rollback:

```bash
# Stop services
sudo systemctl stop nowa-zk-indexer nowa-zk-prover

# Revert to previous version
cd ~/nowa-zk
git log --oneline -5  # Find previous commit hash
git checkout <previous-commit-hash>

# Rebuild
cd indexer && go build -o ../build/indexer-bin ./cmd/indexer && cd ..
cd prover && go build -o ../build/prover-bin ./cmd/prover && cd ..

# Restart
sudo systemctl start nowa-zk-indexer nowa-zk-prover
```

---

## Troubleshooting

### Services won't start after upgrade

```bash
# Check service status
sudo systemctl status nowa-zk-indexer nowa-zk-prover

# View detailed logs
sudo journalctl -u nowa-zk-indexer -n 100
sudo journalctl -u nowa-zk-prover -n 100

# Verify binaries were built
ls -lh ~/nowa-zk/build/
```

### Build failures

```bash
# Clean and rebuild
cd ~/nowa-zk

# Clean Go cache
cd indexer && go clean -cache && cd ..
cd prover && go clean -cache && cd ..

# Try building again
cd indexer && go build -o ../build/indexer-bin ./cmd/indexer && cd ..
cd prover && go build -o ../build/prover-bin ./cmd/prover && cd ..
```

### State corruption

If you suspect state corruption:

```bash
# Stop services
sudo systemctl stop nowa-zk-indexer nowa-zk-prover

# Backup state
sudo cp -r /var/lib/nowa-zk/indexer/state /var/lib/nowa-zk/indexer/state.backup
```

---

## Notes

- **State data** is stored in `/var/lib/nowa-zk/` and is not affected by code updates
- **Binaries** are in `~/nowa-zk/build/` and will be replaced during rebuild
- **Keys** in `/var/lib/nowa-zk/prover/keys/` should not change unless circuit changes
- Always check release notes for breaking changes or special upgrade instructions
