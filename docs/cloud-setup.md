# Cloud Setup Guide

This guide describes how to deploy the **Tan-ZK Sequencer and Prover** on a Linux cloud server.

## Prerequisites

*   Linux server (Ubuntu 20.04+ recommended)
*   `sudo` access
*   Git, Make, curl installed

---

## 1. Directory Setup

```bash
# Create directory for persistent data
sudo mkdir -p /var/lib/tan-zk/sequencer/state
sudo mkdir -p /var/lib/tan-zk/prover/keys
sudo mkdir -p /var/lib/tan-zk/prover/data

# Set ownership to current user
sudo chown -R $USER:$USER /var/lib/tan-zk
```

---

## 2. SSH Key Setup

Generate an SSH key to clone the private repository.

```bash
# Generate SSH key
ssh-keygen -t ed25519 -C "your-server-name"

# Display public key
cat ~/.ssh/id_ed25519.pub
```

*   Add this key to **GitHub Repo Settings** → **Deploy Keys**.

**Test Connection:**
```bash
ssh -T git@github.com
```

---

## 3. Install Dependencies

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install build tools
sudo apt install -y make git build-essential curl

# Install Go 1.23.2
curl -OL https://go.dev/dl/go1.23.2.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz
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

## 4. Clone & Build

```bash
# Clone the repository
git clone git@github.com:tannetwork/tan-zk.git ~/tan-zk
cd ~/tan-zk

# Initialize submodules (if any)
git submodule update --init --recursive
```

### Build Prover Keys

```bash
cd ~/tan-zk/prover

# Build prover binary first
go build -o ../build/prover-bin ./cmd/prover

# Generate keys and verifier contract
../build/prover-bin setup --output-dir ../keys --contract-output ../contracts/src/generated

cd ..
```

### Build Contracts

```bash
cd ~/tan-zk/contracts

# Build all contracts
forge build

cd ..
```

### Build Sequencer

```bash
cd ~/tan-zk/sequencer

# Build sequencer binary
go build -o ../build/sequencer-bin ./cmd/sequencer

cd ..
```

### Persist Keys

```bash
# Copy keys to persistent storage
sudo cp -r ~/tan-zk/keys/* /var/lib/tan-zk/prover/keys/
sudo chown -R $USER:$USER /var/lib/tan-zk
```

---

## 5. Environment Configuration

Create environment configuration file.

```bash
sudo mkdir -p /etc/tan
sudo nano /etc/tan/.env
```

### `/etc/tan/.env`

```bash
# RPC URL (Your blockchain network endpoint)
RPC=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY

# Private Key for deployment and proof submission
PRIVATE_KEY=0xYOUR_PRIVATE_KEY_HERE

# Start indexing from this block
INDEX_FROM_BLOCK=0

# Etherscan API key (for contract verification)
ETHERSCAN_API_KEY=YOUR_ETHERSCAN_KEY

# Server Persistence Paths
STATE_DB_PATH=/var/lib/tan-zk/sequencer/state
```

**Secure the file:**
```bash
sudo chmod 600 /etc/tan/.env
```

---

## 6. Deploy Contracts

```bash
cd ~/tan-zk

# Load environment variables
source /etc/tan/.env

# Deploy contracts
cd contracts
forge script script/Deploy.s.sol --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast

# Optional: Verify on Etherscan
# forge script script/Deploy.s.sol --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast --verify

cd ..
```

**Save the deployed contract address** - you'll need it for the prover service configuration.

---

## 7. Systemd Services

### Sequencer Service

Create `/etc/systemd/system/tan-sequencer.service`:

```bash
sudo nano /etc/systemd/system/tan-sequencer.service
```

```ini
[Unit]
Description=Tan-ZK Sequencer Service
After=network-online.target

[Service]
User=YOUR_USERNAME
Group=YOUR_USERNAME
WorkingDirectory=/home/YOUR_USERNAME/tan-zk
EnvironmentFile=/etc/tan/.env
ExecStart=/home/YOUR_USERNAME/tan-zk/build/sequencer-bin start --rpc-url ${RPC} --state-db-path ${STATE_DB_PATH}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Prover Service

Create `/etc/systemd/system/tan-prover.service`:

```bash
sudo nano /etc/systemd/system/tan-prover.service
```

```ini
[Unit]
Description=Tan-ZK Prover Service
After=network-online.target

[Service]
User=YOUR_USERNAME
Group=YOUR_USERNAME
WorkingDirectory=/home/YOUR_USERNAME/tan-zk
EnvironmentFile=/etc/tan/.env
ExecStart=/home/YOUR_USERNAME/tan-zk/build/prover-bin start --keys-dir /var/lib/tan-zk/prover/keys --contract YOUR_CONTRACT_ADDRESS --private-key ${PRIVATE_KEY}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

> [!IMPORTANT]
> Replace `YOUR_USERNAME` with your actual username and `YOUR_CONTRACT_ADDRESS` with the deployed contract address from step 6.

---

## 8. Start Services

### Enable Services

```bash
sudo systemctl daemon-reload
sudo systemctl enable tan-sequencer tan-prover
```

### Start Sequencer

```bash
sudo systemctl start tan-sequencer
```

**Check Status & Logs:**
```bash
sudo systemctl status tan-sequencer
sudo journalctl -u tan-sequencer -f
# Wait until you see "Sequencer started" or block imports
```

### Start Prover

Once the sequencer is running smoothly:

```bash
sudo systemctl start tan-prover
```

**Check Status & Logs:**
```bash
sudo systemctl status tan-prover
sudo journalctl -u tan-prover -f
```

---

## 9. Verify Deployment

### Check Service Status

```bash
# Check both services
sudo systemctl status tan-sequencer tan-prover

# Follow logs
sudo journalctl -u tan-sequencer -f
sudo journalctl -u tan-prover -f
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
sudo journalctl -u tan-sequencer -n 50
sudo journalctl -u tan-prover -n 50

# Verify binaries exist
ls -lh ~/tan-zk/build/

# Verify permissions
ls -lh /var/lib/tan-zk/
```

### Connection Issues

*   Verify RPC URL is accessible: `curl -X POST $RPC -H "Content-Type: application/json" -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'`
*   Check firewall rules
*   Ensure private key has sufficient funds

### Key Issues

```bash
# Verify keys exist
ls -lh /var/lib/tan-zk/prover/keys/

# Regenerate if needed
cd ~/tan-zk/prover
../build/prover-bin setup --output-dir ../keys --contract-output ../contracts/src/generated
sudo cp -r ../keys/* /var/lib/tan-zk/prover/keys/
```
