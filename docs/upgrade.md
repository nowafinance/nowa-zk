# Cloud Upgrade Guide

This guide describes how to upgrade the **Tan-ZK Sequencer and Prover** running in the **cloud setup** (using `systemd`).

## Prerequisites

*   SSH access to the server.
*   `sudo` privileges.

## Step 1: Stop Services

Stop the running systemd services to ensure no data corruption during the update.

```bash
sudo systemctl stop tan-sequencer tan-prover
```

Verify they are stopped:

```bash
sudo systemctl status tan-sequencer tan-prover
```

## Step 2: Get Latest Code

Navigate to the project directory and pull the latest changes.

```bash
cd ~/tan-zk
git pull origin main
```

## Step 3: Rebuild Binaries

First, regenerate the Swagger documentation to ensure it matches the current environment:

```bash
make swagger swagger-prover
```

Then, recompile the Sequencer and Prover binaries.

> [!WARNING]
> **Do NOT run `make clean`**. While your State DB is strictly persisted in `/var/lib/tan-zk`, running clean removes your `build/` folder and local configs. Just run build.

```bash
make build
```

This updates:
*   `build/sequencer-bin`
*   `build/prover-bin`

## Step 4: Restart Services

Start the services again. They will pick up the newly built binaries automatically.

```bash
sudo systemctl start tan-sequencer tan-prover
```

## Step 5: Verify Upgrade

Check the logs to ensure everything started correctly and is processing blocks.

```bash
# Check Sequencer Logs
sudo journalctl -u tan-sequencer -f

# Check Prover Logs
sudo journalctl -u tan-prover -f
```

## Summary of Commands

```bash
# 1. Stop Services
sudo systemctl stop tan-sequencer tan-prover

# 2. Update Code
cd ~/tan-zk
git pull origin main

# 3. Build
make swagger swagger-prover
make build

# 4. Restart
sudo systemctl start tan-sequencer tan-prover
```
