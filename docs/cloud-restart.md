# Cloud Restart & Reset Guide

This guide provides a script to completely **rebuild**, **redeploy**, **reset the database**, and **restart** the Tan-ZK services on your cloud server.

> [!WARNING]
> This process **deletes all chain data** and generates new cryptographic keys. The chain will start from block 0, and previous transactions will be lost.

## Automated Restart Script

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

# Generate NEW prover keys and verifier contract
echo "🔑 Generating new keys..."
cd prover
go run ./cmd/prover setup --output-dir ../keys --contract-output ../contracts/src/generated
cd ..

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

source /etc/tan/.env

cd contracts
forge script script/Deploy.s.sol --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast
cd ..

# Save deployment info
CHAIN_ID=$(cast chain-id --rpc-url $RPC)
mkdir -p deployments
cp contracts/deployments/$CHAIN_ID.json deployments/deployment.json

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

## Usage

1. Save this script as `restart.sh` in your home directory
2. Make it executable: `chmod +x restart.sh`
3. Run it: `./restart.sh`

> [!CAUTION]
> This script will **permanently delete** all existing chain data. Make sure you have backups if needed.
