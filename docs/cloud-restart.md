# Cloud Restart & Reset Guide

This guide provides a script to completely **rebuild**, **redeploy**, **reset the database**, and **restart** the Tan-ZK services on your cloud server.

> [!WARNING]
> This process **deletes all chain data** and generates new cryptographic keys. The chain will start from block 0, and previous transactions will be lost.

## Automated Restart Script

```sh
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
make clean
make deps
make setup    # Generates NEW keys
make build

# --- 4. Persist New Keys ---
echo "🔑 Updating persistent keys..."
sudo cp -r .tan-zk/keys/* /var/lib/tan-zk/prover/keys/
# Fix permissions so the 'tan' user can read them
sudo chown -R $USER:$USER /var/lib/tan-zk

# --- 5. Redeploy Contracts ---
echo "🚀 Redeploying contracts..."
# Ensure we are using the correct env
if [ ! -f .env ]; then
    ln -sf /etc/tan/.env .env
fi
make deploy

# --- 6. Restart Services ---
echo "✅ Restarting services..."
sudo systemctl daemon-reload
sudo systemctl start tan-sequencer
sudo systemctl start tan-prover

echo "🎉 Reset Complete!"
echo "Check status:"
echo "  sudo systemctl status tan-sequencer"
echo "  sudo systemctl status tan-prover"
```
