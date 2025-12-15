# Troubleshooting Guide

This guide covers common issues you may encounter when running Tan-ZK on a cloud server.

---

## Deployment Issues

### Error: "contract source info format must be `<path>:<contractname>`"

**Symptom:**
```bash
Error: contract source info format must be `<path>:<contractname>` or `<contractname>`
```

**Causes:**
1. Missing contract name in forge script command
2. Environment variables not loaded

**Solution:**

#### 1. Use Correct Command Format

The correct command includes `:Deploy` after the script path:

```bash
forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast
```

#### 2. Verify Environment Variables Are Loaded

```bash
# Load with auto-export
set -a
source /etc/tan/.env
set +a

# Verify they're loaded
echo "RPC: $RPC"
echo "Private Key: ${PRIVATE_KEY:0:10}..."  # Show first 10 chars only
```

#### 3. Check .env File Format

Your `/etc/tan/.env` should look like:

```bash
# NO SPACES around = sign
RPC=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
PRIVATE_KEY=0xYOUR_PRIVATE_KEY_HERE
INDEX_FROM_BLOCK=0
ETHERSCAN_API_KEY=YOUR_KEY
STATE_DB_PATH=/var/lib/tan-zk/sequencer/state
```

**Common mistakes:**
- ❌ `RPC = http://...` (spaces around `=`)
- ❌ `RPC="http://..."` (quotes, not needed)
- ✅ `RPC=http://...` (correct)

---

## ZK Proof Generation Issues

### Error: "constraint #XXXXX is not satisfied: -1 ⋅ 1 != 0"

**Symptom:**
```
ERR error="constraint #113964 is not satisfied: -1 ⋅ 1 != 0" nbConstraints=696402
❌ Failed to generate proof
```

**Cause:**
The prover keys are incompatible with the circuit code. This happens after a code update when the circuit logic changed but keys weren't regenerated.

**Solution:**

You must regenerate keys AND redeploy contracts:

```bash
# 1. Stop services
sudo systemctl stop tan-sequencer tan-prover

# 2. Navigate to project
cd ~/tan-zk
git pull origin main

# 3. Rebuild prover
cd prover
go build -o ../build/prover-bin ./cmd/prover
cd ..

# 4. Regenerate keys (this also regenerates RollupVerifier.sol)
./build/prover-bin setup --output-dir ./keys --contract-output ./contracts/src/generated

# 5. Rebuild contracts
cd contracts
forge build
cd ..

# 6. Redeploy contracts
cd contracts
set -a
source /etc/tan/.env
set +a
forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast
cd ..

# 7. Save the NEW contract address from output

# 8. Copy new keys
sudo cp -r ./keys/* /var/lib/tan-zk/prover/keys/
sudo chown -R $USER:$USER /var/lib/tan-zk

# 9. Update prover service with new contract address
sudo nano /etc/systemd/system/tan-prover.service
# Update: ExecStart=.../prover-bin start --keys-dir ... --contract <NEW_ADDRESS> ...

# 10. Restart
sudo systemctl daemon-reload
sudo systemctl start tan-sequencer tan-prover
```

> [!WARNING]
> This redeploys the contract, losing all previous on-chain state.

### Prover Using Wrong Contract Address

**Symptom:**
```
ℹ️  Auto-loaded Contract: 0xOLD_CONTRACT_ADDRESS
🔄 Resuming from batch #347
❌ Failed to generate proof: constraint #113964 is not satisfied
```

**Cause:**
The prover auto-loads the contract address from `.tan-zk/deployments.json`, but this file has the OLD contract address after you redeployed.

**Solution:**

```bash
# 1. Stop prover
sudo systemctl stop tan-prover

# 2. Find your new deployment file (replace CHAIN_ID with actual value from deployment)
ls -lt ~/tan-zk/contracts/deployments/

# 3. Update deployments.json with new contract
cp ~/tan-zk/contracts/deployments/CHAIN_ID.json ~/tan-zk/.tan-zk/deployments.json

# 4. Verify it updated
cat ~/tan-zk/.tan-zk/deployments.json
# Should show your NEW BatchRegistry address

# 5. Delete prover database to start from batch 0
find ~/tan-zk/.tan-zk/ -name "*.db" -delete
find ~/tan-zk/.tan-zk/ -name "*.bolt" -delete

# 6. Restart prover
sudo systemctl start tan-prover

# 7. Monitor - should see NEW contract address and start from batch 0
sudo journalctl -u tan-prover -f
```

### Prover Resuming from Old Batch (Need to Start from 0)

**Symptom:**
```
🔄 Resuming from batch #347
📦 Loaded last known state root from DB
```

**Cause:**
The prover stores its database in `~/tan-zk/.tan-zk/` directory (usually `prover.db` or `.bolt` files). After redeploying contracts, you need to delete this to start fresh.

**Solution:**

```bash
# Stop prover
sudo systemctl stop tan-prover

# Find and delete prover database files
find ~/tan-zk/.tan-zk/ -name "*.db" -ls
find ~/tan-zk/.tan-zk/ -name "*.bolt" -ls

# Delete them
find ~/tan-zk/.tan-zk/ -name "*.db" -delete
find ~/tan-zk/.tan-zk/ -name "*.bolt" -delete

# Also clear any prover data in /var/lib
sudo rm -rf /var/lib/tan-zk/prover/data/*

# Restart
sudo systemctl start tan-prover
sudo journalctl -u tan-prover -f
# Should see: "No previous state found" or "Starting from batch #0"
```

---

## Service Not Starting

### Sequencer Won't Start

**Check logs:**
```bash
sudo journalctl -u tan-sequencer -n 50
```

**Common issues:**

1. **RPC connection failed**
   ```bash
   # Test RPC manually
   curl -X POST $RPC \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
   ```

2. **State DB permission issues**
   ```bash
   # Fix permissions
   sudo chown -R $USER:$USER /var/lib/tan-zk
   ```

3. **Port already in use**
   ```bash
   # Check what's using port 8080
   sudo lsof -i :8080
   ```

### Prover Won't Start

**Check logs:**
```bash
sudo journalctl -u tan-prover -n 50
```

**Common issues:**

1. **Keys not found**
   ```bash
   # Verify keys exist
   ls -lh /var/lib/tan-zk/prover/keys/
   
   # Should see: rollup.pk and rollup.vk
   ```

2. **Wrong contract address**
   ```bash
   # Check service file
   sudo cat /etc/systemd/system/tan-prover.service | grep contract
   
   # Verify contract exists on chain
   cast code <CONTRACT_ADDRESS> --rpc-url $RPC
   ```

3. **Insufficient funds**
   ```bash
   # Check prover wallet balance
   cast balance <PROVER_ADDRESS> --rpc-url $RPC
   ```

---

## Build Failures

### Go Build Fails

```bash
# Clear Go cache
cd ~/tan-zk
cd sequencer && go clean -cache && go mod tidy && cd ..
cd prover && go clean -cache && go mod tidy && cd ..

# Rebuild
cd sequencer && go build -o ../build/sequencer-bin ./cmd/sequencer && cd ..
cd prover && go build -o ../build/prover-bin ./cmd/prover && cd ..
```

### Forge Build Fails

```bash
# Clear Forge cache
cd ~/tan-zk/contracts
forge clean

# Rebuild
forge build
```

---

## Environment Variable Issues

### Variables Not Persisting

If you `source /etc/tan/.env` but variables disappear:

**Solution 1: Use `set -a`**
```bash
set -a          # Enable auto-export
source /etc/tan/.env
set +a          # Disable auto-export
```

**Solution 2: Export manually**
```bash
export $(grep -v '^#' /etc/tan/.env | xargs)
```

**Solution 3: Use in one line**
```bash
env $(cat /etc/tan/.env | xargs) forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast
```

### Check If Variables Are Set

```bash
# Print all variables
cat /etc/tan/.env

# Test if loaded in current shell
echo "RPC=$RPC"
echo "PRIVATE_KEY=${PRIVATE_KEY:0:10}..."
```

---

## Getting Help

If you encounter an issue not covered here:

1. **Check service logs:**
   ```bash
   sudo journalctl -u tan-sequencer -f
   sudo journalctl -u tan-prover -f
   ```

2. **Check system resources:**
   ```bash
   htop  # CPU and memory usage
   df -h # Disk space
   ```

3. **Verify network connectivity:**
   ```bash
   ping 8.8.8.8
   curl -I https://google.com
   ```

4. **Check all services:**
   ```bash
   sudo systemctl status tan-sequencer tan-prover
   ```
