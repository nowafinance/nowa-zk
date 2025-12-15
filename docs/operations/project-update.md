# Project Update & Full Reset Guide

This guide covers procedures when the ZK-Tan codes changes, most of the time it require key regeneration and contract redeployment again.

---

## When Do You Need This?

Use this guide when:
- ✅ The ZK circuit code has changed (check release notes)
- ✅ You're getting constraint satisfaction errors in the prover
- ✅ Keys need to be regenerated for any reason

> [!CAUTION]
> **Circuit changes require FULL contract redeployment** because the `RollupVerifier.sol` contract is cryptographically tied to the verification key. Old contracts cannot verify proofs from new keys.

---

## Quick Command Reference

```bash
# 1. Stop both services
sudo systemctl stop tan-sequencer tan-prover

# 2. Navigate to project
cd ~/tan-zk
git pull origin main

# 3. Rebuild prover binary
cd prover
go build -o ../build/prover-bin ./cmd/prover
cd ..

# 4. Regenerate keys AND new RollupVerifier.sol
./build/prover-bin setup --output-dir ./keys --contract-output ./contracts/src/generated

# 5. Rebuild contracts (with new verifier)
cd contracts
forge build
cd ..

# 6. Redeploy contracts to chain

# Fix .env permissions (if needed)
sudo chmod 640 /etc/tan/.env
sudo chown $USER:$USER /etc/tan/.env

# Navigate to contracts directory
cd ~/tan-zk/contracts

# Load environment variables
set -a  # Auto-export all variables
source /etc/tan/.env
set +a

# Verify variables are loaded
echo "RPC: $RPC"
echo "Private Key loaded: ${PRIVATE_KEY:0:10}..."

# Deploy with correct contract name
forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast

cd ..
# ⚠️ Note the CHAIN_ID and NEW CONTRACT ADDRESS from deploy output

# 7. Update deployments.json (prover auto-loads from this)
# Replace CHAIN_ID with your actual chain ID from deployment output
cp ~/tan-zk/contracts/deployments/CHAIN_ID.json ~/tan-zk/.tan-zk/deployments.json

# Verify it updated
cat ~/tan-zk/.tan-zk/deployments.json

# 8. Copy new keys to persistent storage
sudo cp -r ./keys/* /var/lib/tan-zk/prover/keys/
sudo chown -R $USER:$USER /var/lib/tan-zk

# 9. Delete prover database to start from batch 0
find ~/tan-zk/.tan-zk/ -name "*.db" -delete
find ~/tan-zk/.tan-zk/ -name "*.bolt" -delete

# 10. Reload systemd and restart both services
sudo systemctl daemon-reload
sudo systemctl start tan-sequencer tan-prover

# 11. Monitor (check that prover uses new contract and starts from batch 0)
sudo journalctl -u tan-prover -f
# You should see: "Auto-loaded Contract: 0x..." (your NEW address)
# And: "Starting from batch #0" or "No previous state found"
```

---

## Full Reset Script (Nuclear Option)

Use this automated script to completely **reset everything** - database, keys, and contracts.

> [!WARNING]
> This **deletes ALL chain data** and starts from block 0. Use only when necessary.

### Automated Reset Script

Save as `~/reset-tan-zk.sh`:

```bash
#!/bin/bash
set -e

echo "⚠️  WARNING: This will delete all chain data, rotate keys, and redeploy contracts."
read -p "Are you sure you want to proceed? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

# --- 1. Stop Services ---
echo "🛑 Stopping services..."
sudo systemctl stop tan-sequencer tan-prover

# --- 2. Cleanup Data (Reset Database) ---
echo "🧹 Cleaning up old data..."
# Clear Sequencer State
if [ -d "/var/lib/tan-zk/sequencer/state" ]; then
    sudo rm -rf /var/lib/tan-zk/sequencer/state/*
    echo "   - Sequencer state cleared"
fi
# Clear Prover Data/Keys
if [ -d "/var/lib/tan-zk/prover" ]; then
    sudo rm -rf /var/lib/tan-zk/prover/keys/*
    sudo rm -rf /var/lib/tan-zk/prover/data/*
    echo "   - Prover data & keys cleared"
fi

# --- 3. Pull & Rebuild ---
echo "🏗️  Rebuilding..."
cd ~/tan-zk
git pull origin main

# Clean old artifacts
cd prover
go clean
cd ../sequencer
go clean
cd ../contracts
forge clean
cd ..

# Build prover binary first
cd prover
go build -o ../build/prover-bin ./cmd/prover
cd ..

# Generate NEW prover keys and verifier contract
echo "🔑 Generating new keys..."
./build/prover-bin setup --output-dir ./keys --contract-output ./contracts/src/generated

# Build contracts
echo "🏗️  Building contracts..."
cd contracts
forge build
cd ..

# Build Go binaries
echo "🏗️  Building binaries..."
cd sequencer
go build -o ../build/sequencer-bin ./cmd/sequencer
cd ../prover
go build -o ../build/prover-bin ./cmd/prover
cd ..

# --- 4. Persist New Keys ---
echo "🔑 Updating persistent keys..."
sudo mkdir -p /var/lib/tan-zk/prover/keys
sudo cp -r keys/* /var/lib/tan-zk/prover/keys/
# Fix permissions so the service user can read them
sudo chown -R $USER:$USER /var/lib/tan-zk

# --- 5. Redeploy Contracts ---
echo "🚀 Redeploying contracts..."
# Load environment variables
if [ ! -f /etc/tan/.env ]; then
    echo "❌ Error: /etc/tan/.env not found!"
    exit 1
fi

set -a
source /etc/tan/.env
set +a

cd contracts
forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast
cd ..

# Save deployment info and update deployments.json
CHAIN_ID=$(cast chain-id --rpc-url $RPC)
cp contracts/deployments/$CHAIN_ID.json .tan-zk/deployments.json
echo "✅ Updated .tan-zk/deployments.json with new contract"

# Delete prover database to start from batch 0
rm -f .tan-zk/*.db .tan-zk/*.bolt
echo "✅ Deleted prover database"

# --- 6. Restart Services ---
echo "✅ Restarting services..."
sudo systemctl daemon-reload
sudo systemctl start tan-sequencer
sudo systemctl start tan-prover

echo "🎉 Reset Complete!"
echo "Check status:"
echo "  sudo systemctl status tan-sequencer"
echo "  sudo systemctl status tan-prover"
echo "  sudo journalctl -u tan-sequencer -f"
echo "  sudo journalctl -u tan-prover -f"
```

### Usage

```bash
# 1. Save the script
nano ~/reset-tan-zk.sh

# 2. Make executable
chmod +x ~/reset-tan-zk.sh

# 3. Run
./reset-tan-zk.sh
```

---

## What Happens During Circuit Update

### 1. Key Regeneration
- New `rollup.pk` (proving key)  
- New `rollup.vk` (verification key)
- New `RollupVerifier.sol` contract generated from verification key

### 2. Contract Redeployment
- Deploys new StateManager
- Deploys new Verifier (from the new verification key)
- Deploys new BatchRegistry
- **All previous on-chain state is lost**

### 3. Service Configuration
- Prover must point to the NEW contract address
- Sequencer continues with same config
- Keys must be copied to persistent storage

---

## Troubleshooting

### "Constraint not satisfied" error after update

**Cause:** Keys don't match the circuit  
**Solution:** Follow the full circuit update procedure above

### Old contract address in systemd service

```bash
# Check current config
sudo cat /etc/systemd/system/tan-prover.service | grep contract

# Edit with new address
sudo nano /etc/systemd/system/tan-prover.service

# Reload
sudo systemctl daemon-reload
sudo systemctl restart tan-prover
```

### Keys not found

```bash
# Verify keys exist
ls -lh /var/lib/tan-zk/prover/keys/

# Should see: rollup.pk and rollup.vk

# If missing, regenerate
cd ~/tan-zk
./build/prover-bin setup --output-dir ./keys --contract-output ./contracts/src/generated
sudo cp -r ./keys/* /var/lib/tan-zk/prover/keys/
```

---

## See Also

- [Upgrade Guide](upgrade.md) - For regular code updates without circuit changes
- [Troubleshooting Guide](troubleshooting.md) - Common issues and solutions
