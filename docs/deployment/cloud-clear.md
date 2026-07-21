# Cleanup Guide

```bash
# Stop and disable services
sudo systemctl stop nowa-indexer nowa-prover
sudo systemctl disable nowa-indexer nowa-prover

# Remove service files
sudo rm -f /etc/systemd/system/nowa-indexer.service
sudo rm -f /etc/systemd/system/nowa-prover.service
sudo systemctl daemon-reload

# Remove data directories
sudo rm -rf /var/lib/nowa-zk

# Remove deployment info
rm -rf ~/.nowa-zk

# Find and remove ALL database files (indexer and prover)
# The indexer stores data in ~/.nowa-zk/indexer/data by default
rm -rf ~/.nowa-zk/indexer/
rm -rf ~/.nowa-zk/prover/

# Also check for databases in other common locations
find ~/ -type d \( -name "*indexer-db*" -o -name "*prover-db*" -o -name ".badger*" \) -exec rm -rf {} + 2>/dev/null

# Clean build artifacts in repository
cd ~/nowa-zk
rm -rf build/
rm -rf keys/
rm -rf contracts/out/
rm -rf contracts/cache/
rm -rf contracts/broadcast/
rm -rf contracts/deployments/
rm -rf indexer/.indexer-db/
rm -rf prover/.prover-db/
rm -rf .indexer-db/
rm -rf .prover-db/

# Clean Go caches
go clean -modcache
rm -rf ~/.cache/go-build

# Clean Foundry caches
rm -rf ~/.foundry/cache

echo "✅ Cleanup complete! Source code and .env preserved"
```

## What's Removed

- ✅ Services and systemd files
- ✅ Database and state data
- ✅ Built binaries
- ✅ Generated keys
- ✅ Contract artifacts
- ✅ Build caches

## What's Preserved

- ❌ Source code (`~/nowa-zk` repository)
- ❌ `.env` file (`/etc/nowa-zk/.env`)

## Verification

```bash
# Check services are stopped
sudo systemctl status nowa-indexer nowa-prover

# Check data removed
ls -la /var/lib/ | grep nowa-zk  # Should be empty
ls -la ~/.nowa-zk                # Should not exist

# Check source code preserved
ls -la ~/nowa-zk                  # Should still exist
```

## Redeploy

After cleanup, redeploy by following the [deployment guide](./cloud.md) from **Step 4** (contracts are already built, just need to regenerate keys and redeploy).
