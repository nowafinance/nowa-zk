# Cleanup Guide

Remove all deployment data and build artifacts while preserving source code and `.env` configuration.

## Complete Cleanup

```bash
# Stop and disable services
sudo systemctl stop nowa-zk-sequencer nowa-zk-prover
sudo systemctl disable nowa-zk-sequencer nowa-zk-prover

# Remove service files
sudo rm -f /etc/systemd/system/nowa-zk-sequencer.service
sudo rm -f /etc/systemd/system/nowa-zk-prover.service
sudo systemctl daemon-reload

# Remove data directories
sudo rm -rf /var/lib/nowa-zk

# Remove deployment info
rm -rf ~/.nowa-zk

# Find and remove ALL database files (sequencer and prover)
# The sequencer stores data in ~/.nowa-zk/sequencer/data by default
rm -rf ~/.nowa-zk/sequencer/
rm -rf ~/.nowa-zk/prover/

# Also check for databases in other common locations
find ~/ -type d \( -name "*sequencer-db*" -o -name "*prover-db*" -o -name ".badger*" \) -exec rm -rf {} + 2>/dev/null

# Clean build artifacts in repository
cd ~/nowa-zk
rm -rf build/
rm -rf keys/
rm -rf contracts/out/
rm -rf contracts/cache/
rm -rf contracts/broadcast/
rm -rf contracts/deployments/
rm -rf sequencer/.sequencer-db/
rm -rf prover/.prover-db/
rm -rf .sequencer-db/
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
sudo systemctl status nowa-zk-sequencer nowa-zk-prover

# Check data removed
ls -la /var/lib/ | grep Nowa-ZK  # Should be empty
ls -la ~/.Nowa-ZK                 # Should not exist

# Check source code preserved
ls -la ~/Nowa-ZK                  # Should still exist
```

## Redeploy

After cleanup, redeploy by following the [deployment guide](./cloud.md) from **Step 4** (contracts are already built, just need to regenerate keys and redeploy).

---

## Nuclear Cleanup (If Above Doesn't Work)

If the sequencer still resumes from old block numbers, use this aggressive cleanup:

```bash
# Stop services
sudo systemctl stop nowa-zk-sequencer nowa-zk-prover

# Remove EVERYTHING in home directory with these names
find ~/ -name "*nowa-zk*" -type d ! -path "*/nowa-zk/.git/*" ! -path "*/nowa-zk" -exec rm -rf {} + 2>/dev/null
find ~/ -name "*badger*" -type d -exec rm -rf {} + 2>/dev/null
find ~/ -name "*sequencer*" -type d ! -path "*/nowa-zk/*" -exec rm -rf {} + 2>/dev/null
find ~/ -name "*prover*" -type d ! -path "*/nowa-zk/*" -exec rm -rf {} + 2>/dev/null

# Explicitly remove known locations
rm -rf ~/.nowa-zk
rm -rf /var/lib/nowa-zk
sudo rm -rf /var/lib/nowa-zk

# Verify
echo "Checking for remaining database files..."
find ~/ -name "*badger*" -o -name "*nowa-zk*" 2>/dev/null | grep -v "nowa-zk/.git" | grep -v "/nowa-zk$"
```

If the above find command shows NO output, databases are gone! ✅
