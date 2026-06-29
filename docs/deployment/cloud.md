# Cloud Setup Guide

This guide describes how to deploy the **Nowa-ZK Sequencer and Prover** on a Linux cloud server.

## Prerequisites

*   Linux server (Ubuntu 20.04+ recommended)
*   `sudo` access
*   Git, Make, curl installed

---

## 1. SSH Key Setup

Generate an SSH key to clone the private repository.

```bash
# Generate SSH key
ssh-keygen -t ed25519 -C "zkprover"

# Display public key
cat ~/.ssh/id_ed25519.pub
```

*   Add this key to **GitHub Repo Settings** → **Deploy Keys**.

**Test Connection:**
```bash
ssh -T git@github.com
```

---

## 2. Install Dependencies

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install build tools
sudo apt install -y make git build-essential curl

# Install Go 1.24.10
curl -OL https://go.dev/dl/go1.24.10.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.10.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Verify Go installation
go version

# Install Foundry
curl -L https://foundry.paradigm.xyz | bash
source ~/.bashrc
foundryup

# Verify Foundry installation
forge --version
cast --version
```

---

## 3. Clone & Build

```bash
# Clone the repository
git clone git@github.com:tannetwork/nowa-zk.git ~/nowa-zk
cd ~/nowa-zk

# Initialize submodules (if any)
git submodule update --init --recursive
```

### Build Prover Keys

```bash
cd ~/nowa-zk/prover

# Build prover binary first
go build -o ../build/prover-bin ./cmd/prover

# Generate keys and verifier contract
../build/prover-bin setup --output-dir ../keys --contract-output ../contracts/src/generated

cd ..
```

### Build Contracts

```bash
cd ~/nowa-zk/contracts

# Build all contracts
forge build

cd ..
```

### Build Sequencer

```bash
cd ~/nowa-zk/sequencer

# Build sequencer binary
go build -o ../build/sequencer-bin ./cmd/sequencer

cd ..
```

## 4. Directory Setup

```bash
# Create directory for persistent data
sudo mkdir -p /var/lib/nowa-zk/sequencer/state
sudo mkdir -p /var/lib/nowa-zk/prover/keys
sudo mkdir -p /var/lib/nowa-zk/prover/data

# Set ownership to current user
sudo chown -R $USER:$USER /var/lib/nowa-zk
```

### Persist Keys Copy

```bash
# Copy keys to persistent storage
sudo cp -r ~/nowa-zk/keys/* /var/lib/nowa-zk/prover/keys/
sudo chown -R $USER:$USER /var/lib/nowa-zk
```

---


---

## 5. Environment Configuration

Create environment configuration file.

```bash
sudo mkdir -p /etc/nowa
sudo nano /etc/nowa/.env
```

### `/etc/nowa/.env`

```bash
RPC_SEQUENCER=http://0.0.0.0:8545
RPC_PROVER=https://ethereum-sepolia-rpc.publicnode.com
PRIVATE_KEY=0xYOUR_PRIVATE_KEY_HERE
TRAFFIC_GEN_KEY=0xYOUR_TRAFFIC_GEN_PRIVATE_KEY_HERE
INDEX_FROM_BLOCK=0
PROVER_API=http://0.0.0.0:8081
STATE_DB_PATH=/var/lib/nowa-zk/sequencer/state
```


**Secure the file:**
```bash
sudo chmod 600 /etc/nowa/.env
```

---

## 6. Deploy Contracts

```bash
# Make it readable by owner and group
sudo chmod 640 /etc/nowa/.env

# Change ownership to your user
sudo chown $USER:$USER /etc/nowa/.env

# Now you can read it
set -a
source /etc/nowa/.env
set +a

# Navigate to contracts directory
cd ~/nowa-zk/contracts

# Create deployments directory (required for saving deployment addresses)
mkdir -p deployments

# Deploy to L1 (Sepolia/Mainnet)
forge script script/Deploy.s.sol:Deploy --rpc-url $RPC_PROVER --private-key $PRIVATE_KEY --broadcast

# Optional: Deploy+Verify contracts on block explorer
# forge script script/Deploy.s.sol:Deploy --rpc-url $RPC_PROVER --private-key $PRIVATE_KEY --broadcast --verify --etherscan-api-key $ETHERSCAN_API_KEY

cd ..
```



> [!NOTE]
> **Expected Warning**: You may see `Warning: Script contains a transaction to 0x... which does not contain any code.`
> 
> This is normal! The deployment script sets your deployer address as the authorized sequencer. Since your address is an EOA (wallet), not a contract, it has no code. Just type `yes` to continue.

**Save the deployed contract address** - you'll need it for the prover service configuration.


## Manual Verification on Block Explorer (Optional)

If you deployed the contracts and want to manually verify them on a block explorer, you will need the contract addresses, source code, and compiler settings.

### 1. Find Your Contract Addresses
All deployed contract addresses are automatically saved in your deployments file:
```bash
cat ~/nowa-zk/contracts/deployments/deployments.json
```
sample output 
```
{
  "BatchRegistry": "0x6D2B9e370832A54ea26402963d1D7fA9998d6aFA",
  "GnarkVerifier": "0xff1F31b0c8Af5D8db25734044639316Dfe1513e4",
  "InitialStateRoot": "0x0000000000000000000000000000000000000000000000000000000000000001",
  "Sequencer": "0x8AA96ceA21f85b3b83E9FC5dE7e9Cc53223634D9",
  "StateManager": "0xc0f40402F7ea6140B32871aA2091fe1FcDDc8dF3",
  "VerifierAdapter": "0xa70b0f3D9976D3B8Cf0ECFf28D282e72d5dB6ac8"
}
```

### 2. Compiler Settings
When manually verifying via a web interface, use the following settings:
- **Compiler Type**: Solidity (Single file)
- **Compiler Version**: Run the following command to check the exact version used in your configuration:
  ```bash
  cd ~/nowa-zk/contracts && forge config | grep solc
  ```
- **Optimization**: Check your `foundry.toml` (Default is typically Yes, 200 runs)

### 3. Verification Methods

**Method A: Using Forge (Recommended)**

You can verify all your deployed contracts directly from the terminal. This avoids UI bugs and handles multi-file contracts perfectly.

First, set your endpoints (replace with your actual RPC and Explorer API if different):
```bash
cd ~/nowa-zk/contracts
export VERIFY_RPC="https://node1.nowa.finance"
export VERIFY_API="https://apiexplorer.nowa.finance/api\?"
```

Then, run these commands, replacing the `0x...` placeholders with the exact addresses from your `deployments.json`:

```bash
# 1. Verify StateManager
forge verify-contract <STATE_MANAGER_ADDRESS> src/StateManager.sol:StateManager \
  --rpc-url $VERIFY_RPC --verifier blockscout --verifier-url $VERIFY_API

# 2. Verify GnarkVerifier
forge verify-contract <GNARK_VERIFIER_ADDRESS> src/generated/RollupVerifier.sol:Verifier \
  --rpc-url $VERIFY_RPC --verifier blockscout --verifier-url $VERIFY_API

# 3. Verify VerifierAdapter
forge verify-contract <VERIFIER_ADAPTER_ADDRESS> src/VerifierAdapter.sol:VerifierAdapter \
  --rpc-url $VERIFY_RPC --verifier blockscout --verifier-url $VERIFY_API

# 4. Verify BatchRegistry
forge verify-contract <BATCH_REGISTRY_ADDRESS> src/BatchRegistry.sol:BatchRegistry \
  --rpc-url $VERIFY_RPC --verifier blockscout --verifier-url $VERIFY_API
```

**Method B: Flattening Source Code for Web UI**

If the explorer requires a single flattened file, generate it using:
```bash
cd ~/nowa-zk/contracts
forge flatten src/BatchRegistry.sol > BatchRegistry_flattened.sol
```
*(You can repeat this for `src/StateManager.sol` and the others if needed).*

---

## 7. Configure Deployment Info

The prover service needs to know the deployed contract address. After deployment, the addresses are saved in the Foundry broadcast file.

```bash
# Create the .Nowa-ZK directory
mkdir -p ~/.nowa-zk

# Copy the deployment file from Foundry's broadcast directory
cp ~/nowa-zk/contracts/deployments/deployments.json ~/.nowa-zk/deployments.json

# Verify the file was copied correctly
cat ~/.nowa-zk/deployments.json
```

> [!IMPORTANT]
> The prover auto-loads the `BatchRegistry` contract address from this file. If you redeploy contracts, you must update this file with the new addresses.

---

## 8. Systemd Services

First, set your username (this only needs to be done once):

```bash
# Set your username here (e.g., prover, tan, ubuntu, etc.)
USERNAME=prover
```

### Sequencer Service

Create the sequencer service file:

```bash
sudo tee /etc/systemd/system/nowa-sequencer.service > /dev/null <<EOF
[Unit]
Description=Nowa-ZK Sequencer Service
After=network-online.target

[Service]
User=$USERNAME
Group=$USERNAME
WorkingDirectory=/home/$USERNAME/nowa-zk
EnvironmentFile=/etc/nowa/.env
ExecStart=/home/$USERNAME/nowa-zk/build/sequencer-bin start
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

### Prover Service

Create the prover service file:

```bash
sudo tee /etc/systemd/system/nowa-prover.service > /dev/null <<EOF
[Unit]
Description=Nowa-ZK Prover Service
After=network-online.target

[Service]
User=$USERNAME
Group=$USERNAME
WorkingDirectory=/home/$USERNAME/nowa-zk
EnvironmentFile=/etc/nowa/.env
ExecStart=/home/$USERNAME/nowa-zk/build/prover-bin start --keys-dir /var/lib/nowa-zk/prover/keys
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF
```

---

## 9. Start Services

### Enable Services

```bash
sudo systemctl daemon-reload
sudo systemctl enable nowa-sequencer nowa-prover
```

### Start Sequencer

```bash
sudo systemctl start nowa-sequencer
```

**Check Status & Logs:**
```bash
sudo systemctl status nowa-sequencer
sudo journalctl -u nowa-sequencer -f
```

### Start Prover

Once the sequencer is running smoothly:

```bash
sudo systemctl start nowa-prover
```

**Check Status & Logs:**
```bash
sudo systemctl status nowa-prover
sudo journalctl -u nowa-prover -f
```

---

## 10. Verify Deployment

### Check Service Status

```bash
# Check both services
sudo systemctl status nowa-sequencer nowa-prover

# Follow logs
sudo journalctl -u nowa-sequencer -f
sudo journalctl -u nowa-prover -f
```

### Test API Endpoints

```bash
# Check sequencer status
curl http://localhost:8080/status

# Check latest batch
curl http://localhost:8080/batch/latest
```

---

## Troubleshooting

### Services Won't Start

```bash
# Check logs
sudo journalctl -u nowa-sequencer -n 50
sudo journalctl -u nowa-prover -n 50

# Verify binaries exist
ls -lh ~/nowa-zk/build/

# Verify permissions
ls -lh /var/lib/nowa-zk/
```

### Connection Issues

*   Verify RPC URL is accessible: `curl -X POST $RPC -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'`
*   Check firewall rules
*   Ensure private key has sufficient funds

### Key Issues

```bash
# Verify keys exist
ls -lh /var/lib/nowa-zk/prover/keys/

# Regenerate if needed
cd ~/nowa-zk/prover
../build/prover-bin setup --output-dir ../keys --contract-output ../contracts/src/generated
sudo cp -r ../keys/* /var/lib/nowa-zk/prover/keys/
```

---

## Disk Space Management

Over time, system logs can consume significant disk space. This section covers how to manage disk usage and keep your server healthy.

### Capping journald Logs

Configure systemd-journald to limit log storage to **500 MB**:

```bash
sudo nano /etc/systemd/journald.conf
```

Add or modify these settings:

```ini
[Journal]
SystemMaxUse=500M
SystemKeepFree=1G
```

Apply the changes:

```bash
sudo systemctl restart systemd-journald
```

**Verify the configuration:**

```bash
# Check current log usage
journalctl --disk-usage

# Confirm limits are applied
systemctl show systemd-journald | grep -E 'SystemMaxUse|SystemKeepFree'
```

### What This Fixes

✅ journald logs are capped at **500 MB**  
✅ Disk will not silently fill again  
✅ System remains fully debuggable  
✅ Nowa-ZK services unaffected  

---

## Optional Optimizations

### Reclaim Go Build Cache (Go Developers)

Free up ~2.4 GB by cleaning Go caches:

```bash
go clean -modcache
rm -rf ~/.cache/go-build
```

### Nowa-ZK Disk Strategy

Your sequencer state (`/var/lib/nowa-zk`) will grow over time. Options:

| Strategy | Description |
|----------|-------------|
| Periodic cleanup | Clear old data for dev environments |
| Move to another disk | Relocate `/var/lib/nowa-zk` to a larger partition |
| Bind-mount to SSD | Mount external storage for performance |

### LVM Expansion (Future-Proofing)

If your system uses LVM (`ubuntu-vg/ubuntu-lv`) and has unallocated space, you can extend `/` **without reinstalling**:

```bash
# Check available space
sudo vgdisplay

# Extend the logical volume (example: add 10G)
sudo lvextend -L +10G /dev/ubuntu-vg/ubuntu-lv

# Resize the filesystem
sudo resize2fs /dev/ubuntu-vg/ubuntu-lv
```

---

## System Health Checklist

| Resource  | Status        |
|-----------|---------------|
| RAM       | 🟢 Excellent  |
| Disk      | 🟢 Healthy    |
| Logs      | 🟢 Controlled |
| Stability | 🟢 Solid      |
