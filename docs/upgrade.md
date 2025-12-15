# Cloud Upgrade Guide

This guide describes how to upgrade the **Tan-ZK Sequencer and Prover** running on a cloud server with systemd services.

> [!IMPORTANT]
> This upgrade process **preserves all chain data**. The sequencer state and prover data will remain intact.

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
sudo systemctl stop tan-sequencer tan-prover
```

Verify they are stopped:

```bash
sudo systemctl status tan-sequencer tan-prover
```

---

### Step 2: Get Latest Code

Navigate to the project directory and pull the latest changes.

```bash
cd ~/tan-zk
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
cd ~/tan-zk

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
source /etc/tan/.env
forge script script/Deploy.s.sol --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast
cd ..
# ⚠️ SAVE THE NEW CONTRACT ADDRESS from deploy output

# 5. Copy new keys to persistent storage
sudo cp -r ./keys/* /var/lib/tan-zk/prover/keys/
sudo chown -R $USER:$USER /var/lib/tan-zk

# 6. Update prover systemd service with NEW contract address
sudo nano /etc/systemd/system/tan-prover.service
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
cd ~/tan-zk/contracts

# If contracts changed:
# forge build

cd ..
```

#### Rebuild Sequencer

```bash
cd ~/tan-zk/sequencer

# Rebuild sequencer binary
go build -o ../build/sequencer-bin ./cmd/sequencer

cd ..
```

#### Rebuild Prover

```bash
cd ~/tan-zk/prover

# Rebuild prover binary
go build -o ../build/prover-bin ./cmd/prover

cd ..
```

---

### Step 4: Restart Services

Start the services again. They will pick up the newly built binaries automatically.

```bash
sudo systemctl start tan-sequencer tan-prover
```

---

### Step 5: Verify Upgrade

Check the logs to ensure everything started correctly and is processing blocks.

```bash
# Check Sequencer Logs
sudo journalctl -u tan-sequencer -f

# Check Prover Logs (in another terminal)
sudo journalctl -u tan-prover -f
```

**Look for:**
- ✅ "Sequencer started" or similar startup message
- ✅ Block processing continuing from last height
- ✅ No error messages
- ✅ Prover connecting and processing batches

---

## Quick Reference

```bash
# Full upgrade workflow
sudo systemctl stop tan-sequencer tan-prover
cd ~/tan-zk
git pull origin main

# Rebuild binaries
cd sequencer && go build -o ../build/sequencer-bin ./cmd/sequencer && cd ..
cd prover && go build -o ../build/prover-bin ./cmd/prover && cd ..

# Restart services
sudo systemctl start tan-sequencer tan-prover

# Monitor
sudo journalctl -u tan-sequencer -f
```

---

## Rollback

If the upgrade fails, you can rollback:

```bash
# Stop services
sudo systemctl stop tan-sequencer tan-prover

# Revert to previous version
cd ~/tan-zk
git log --oneline -5  # Find previous commit hash
git checkout <previous-commit-hash>

# Rebuild
cd sequencer && go build -o ../build/sequencer-bin ./cmd/sequencer && cd ..
cd prover && go build -o ../build/prover-bin ./cmd/prover && cd ..

# Restart
sudo systemctl start tan-sequencer tan-prover
```

---

## Troubleshooting

### Services won't start after upgrade

```bash
# Check service status
sudo systemctl status tan-sequencer tan-prover

# View detailed logs
sudo journalctl -u tan-sequencer -n 100
sudo journalctl -u tan-prover -n 100

# Verify binaries were built
ls -lh ~/tan-zk/build/
```

### Build failures

```bash
# Clean and rebuild
cd ~/tan-zk

# Clean Go cache
cd sequencer && go clean -cache && cd ..
cd prover && go clean -cache && cd ..

# Try building again
cd sequencer && go build -o ../build/sequencer-bin ./cmd/sequencer && cd ..
cd prover && go build -o ../build/prover-bin ./cmd/prover && cd ..
```

### State corruption

If you suspect state corruption:

```bash
# Stop services
sudo systemctl stop tan-sequencer tan-prover

# Backup state
sudo cp -r /var/lib/tan-zk/sequencer/state /var/lib/tan-zk/sequencer/state.backup

# Check state DB integrity (implementation specific)
# If corrupted, restore from backup or resync
```

---

## Notes

- **State data** is stored in `/var/lib/tan-zk/` and is not affected by code updates
- **Binaries** are in `~/tan-zk/build/` and will be replaced during rebuild
- **Keys** in `/var/lib/tan-zk/prover/keys/` should not change unless circuit changes
- Always check release notes for breaking changes or special upgrade instructions
