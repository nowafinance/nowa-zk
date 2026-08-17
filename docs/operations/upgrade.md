# Cloud Upgrade Guide

Upgrade the Sequencer + Prover on a cloud server running as systemd services, **for code
changes that don't touch the ZK circuit**. If the circuit changed, use
[project-update.md](./project-update.md) instead — this guide assumes keys and deployed
contracts stay as-is.

> [!IMPORTANT]
> This process preserves Sequencer state (`~/.nowa-zk/sequencer/nowa_state_db`) and the
> Prover's checkpoint (`~/.nowa-zk/prover/data`).

## Prerequisites
- SSH access, `sudo`
- Services running via systemd (`nowa-sequencer`, `nowa-prover` — see [cloud.md](../deployment/cloud.md))

---

## Steps

### 1. Stop services
```bash
sudo systemctl stop nowa-sequencer nowa-prover
sudo systemctl status nowa-sequencer nowa-prover   # confirm stopped
```

### 2. Get the latest code
```bash
cd ~/nowa-zk
git pull origin main
```

### 3. Check whether the circuit changed

```bash
git diff HEAD@{1} -- prover/circuits/
```
**If non-empty, stop here and follow [project-update.md](./project-update.md) instead**
— rebuilding binaries alone isn't enough, you need new keys and a redeploy.

### 4. Rebuild binaries (circuit unchanged)
```bash
make build
```

### 5. Restart services
```bash
sudo systemctl start nowa-sequencer

# confirm it's healthy before starting the Prover
curl http://localhost:8080/batch/count

sudo systemctl start nowa-prover
```

### 6. Verify

```bash
sudo journalctl -u nowa-sequencer -f
sudo journalctl -u nowa-prover -f
```
Look for:
- `Sequencer API listening on :8080...`
- No repeated `constraint`/`witness` errors from the Prover (that would mean the
  circuit *did* change and step 3 missed it, or the deployed contract is stale — see
  [Troubleshooting](./troubleshooting.md))
- Prover picking up and processing batches without error

---

## Quick Reference

```bash
sudo systemctl stop nowa-sequencer nowa-prover
cd ~/nowa-zk && git pull origin main
git diff HEAD@{1} -- prover/circuits/   # empty? proceed. non-empty? use project-update.md
make build
sudo systemctl start nowa-sequencer nowa-prover
sudo journalctl -u nowa-prover -f
```

---

## Rollback

```bash
sudo systemctl stop nowa-sequencer nowa-prover
cd ~/nowa-zk
git log --oneline -5
git checkout <previous-commit-hash>
make build
sudo systemctl start nowa-sequencer nowa-prover
```
If the commit you're rolling back past touched the circuit, keys/contracts need to roll
back too — that's a [project-update.md](./project-update.md) situation, not a plain
rollback.

---

## Troubleshooting

### Services won't start after upgrade
```bash
sudo systemctl status nowa-sequencer nowa-prover
sudo journalctl -u nowa-sequencer -n 100
sudo journalctl -u nowa-prover -n 100
ls -lh ~/nowa-zk/build/
```

### Build failures
```bash
cd sequencer && go clean -cache && cd ..
cd prover && go clean -cache && cd ..
make build
```

## Notes
- **State** lives entirely under `~/.nowa-zk/` and isn't touched by a code-only upgrade.
- **Binaries** are in `~/nowa-zk/build/` and get replaced by `make build`.
- **Keys** in `~/.nowa-zk/keys/` should only change when the circuit changes — see
  [project-update.md](./project-update.md).
- Always diff `prover/circuits/` before assuming a code upgrade is "just" a rebuild.
