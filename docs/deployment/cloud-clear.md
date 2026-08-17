# Cleanup Guide

Tear down a cloud deployment set up per [cloud.md](./cloud.md).

```bash
# Stop and disable services
sudo systemctl stop nowa-sequencer nowa-prover
sudo systemctl disable nowa-sequencer nowa-prover

# Remove service files
sudo rm -f /etc/systemd/system/nowa-sequencer.service
sudo rm -f /etc/systemd/system/nowa-prover.service
sudo systemctl daemon-reload

# Remove all Nowa-ZK state, keys, and deployment records
rm -rf ~/.nowa-zk

# Clean build artifacts in the repo
cd ~/nowa-zk
rm -rf build/
rm -rf contracts/out/ contracts/cache/ contracts/broadcast/ contracts/deployments/

# Clean Go / Foundry caches
go clean -modcache
rm -rf ~/.cache/go-build
rm -rf ~/.foundry/cache

echo "✅ Cleanup complete! Source code and .env preserved"
```

Everything Nowa-ZK writes at runtime lives under `~/.nowa-zk/` (keys, Sequencer LevelDB
state, Prover BadgerDB checkpoint, `deployments.json`) — there's no separate
`/var/lib/nowa-zk` or `/etc/nowa-zk` to also clean up, unlike an earlier version of this
guide assumed.

## What's Removed
- ✅ systemd services
- ✅ All Sequencer/Prover state and keys (`~/.nowa-zk/`)
- ✅ Built binaries and contract artifacts
- ✅ Build caches

## What's Preserved
- ❌ Source code (`~/nowa-zk`)
- ❌ `.env` (repo root)

## Verification

```bash
sudo systemctl status nowa-sequencer nowa-prover   # should show "not found" / inactive
ls ~/.nowa-zk                                       # should not exist
ls ~/nowa-zk                                        # should still exist
```

## Redeploy

After a full `clean-global`-style wipe, keys are gone too — start from
[cloud.md §3](./cloud.md#3-clone-set-up-keys-build) (`make setup`) onward. If keys are
still intact and you only need a fresh contract + Sequencer state, use the shorter
recipe at [cloud.md §9 "Restart After Clearing All Data"](./cloud.md#9-restart-after-clearing-all-data) instead.
