# Cloud Setup Guide (Root User)

This guide describes how to deploy the **Tan-ZK Sequencer and Prover** on a Linux server assuming a **single user (root)** setup. 

This setup assumes:
*   **User**: `tan` (Current User)
*   **Repo Path**: `/home/tan/tan-zk`
*   **Persistence**: `/var/lib/tan-zk`

---

## 1. Directory Setup

```bash
# Create directory for persistent data
sudo mkdir -p /var/lib/tan-zk/sequencer/state
sudo mkdir -p /var/lib/tan-zk/prover/keys
sudo mkdir -p /var/lib/tan-zk/prover/data

# Set ownership to current user (so service can access it)
sudo chown -R $USER:$USER /var/lib/tan-zk
```

## 2. SSH Key Setup

Generate an SSH key to clone the private repo.

```bash
# 1. Generate SSH key
ssh-keygen -t ed25519 -C "root-server"

# 2. Add this key to GitHub
cat ~/.ssh/id_ed25519.pub
```
*   Add to **GitHub Repo Settings** -> **Deploy Keys**.

**Test Connection:**
```bash
ssh -T git@github.com
```

---

## 3. Clone & Build

```bash
# 1. Clone the repository
git clone git@github.com:tannetwork/tan-zk.git ~/tan-zk
cd ~/tan-zk

# 2. Initialize submodules
git submodule update --init --recursive

# 3. Install dependencies
apt update && apt install -y make git build-essential curl

# 4. Install Go 1.23.2
curl -OL https://go.dev/dl/go1.23.2.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.23.2.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# 5. Install Foundry
curl -L https://foundry.paradigm.xyz | bash
source ~/.bashrc
foundryup

# 6. Run Setup (Generates Keys)
make setup

# 7. Persist Keys
cp -r .tan-zk/keys/* /var/lib/tan-zk/prover/keys/

# 8. Build
make build
```

---

## 4. Environment Configuration

We will store configuration in `/etc/tan/`.

```bash
mkdir -p /etc/tan
sudo nano /etc/tan/.env
sudo chmod 600 /etc/tan/.env
```
### `/etc/tan/.env`
```bash
# Core Configuration
RPC=http://localhost:8545
PRIVATE_KEY=0x...
INDEX_FROM_BLOCK=0

# Server Persistence Overrides
STATE_DB_PATH=/var/lib/tan-zk/sequencer/state
```

## 5. Deploy Contracts

Contracts must be deployed to the target chain (L2) so the Prover knows where to submit proofs.

```bash
# 1. Symlink config to project root (so Make/Foundry can find it)
ln -sf /etc/tan/.env ~/tan-zk/.env

# 2. Deploy Contracts
cd ~/tan-zk
make deploy
```
*   **Result**: Creates `.tan-zk/deployments.json` containing the contract addresses.

---

## 6. Systemd Services (Root)

### Sequencer: `/etc/systemd/system/tan-sequencer.service`

```bash
sudo nano /etc/systemd/system/tan-sequencer.service
```

```ini
[Unit]
Description=Tan-ZK Sequencer Service
After=network-online.target

[Service]
User=tan
Group=tan
WorkingDirectory=/home/tan/tan-zk
EnvironmentFile=/etc/tan/.env
ExecStart=/home/tan/tan-zk/build/sequencer-bin start --rpc-url ${RPC}
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### Prover: `/etc/systemd/system/tan-prover.service`

```bash
sudo nano /etc/systemd/system/tan-prover.service
```

```ini
[Unit]
Description=Tan-ZK Prover Service
After=network-online.target

[Service]
User=tan
Group=tan
WorkingDirectory=/home/tan/tan-zk
EnvironmentFile=/etc/tan/.env
ExecStart=/home/tan/tan-zk/build/prover-bin start --keys-dir /var/lib/tan-zk/prover/keys
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

---

## 7. Start Services

### 1. Reload & Enable
> [!NOTE]
> If you edit the `.service` files later, you MUST run `sudo systemctl daemon-reload` before restarting the services.

```bash
sudo systemctl daemon-reload
sudo systemctl enable tan-sequencer tan-prover
```

### 2. Start Sequencer
```bash
sudo systemctl start tan-sequencer
```

**Check Status & Logs:**
```bash
sudo systemctl status tan-sequencer
sudo journalctl -u tan-sequencer -f
# Wait until you see "Sequencer started" or block imports
```

### 3. Start Prover
Once the sequencer is running smoothly:
```bash
sudo systemctl start tan-prover
```

**Check Status & Logs:**
```bash
sudo systemctl status tan-prover
sudo journalctl -u tan-prover -f
```
