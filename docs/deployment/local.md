# Local Development Setup

This guide describes how to set up the **Nowa-ZK Indexer and Prover** locally for development and testing purposes.

## Prerequisites

*   **Go** 1.24.10+
*   **Foundry** (Forge, Cast, Anvil)
*   **Docker** (Optional, for containerized testing)
*   `make`, `git`, `build-essential`, `curl`

---

## 1. Install Dependencies

If you haven't installed the required tools yet:

```bash
sudo apt update
sudo apt install -y make git build-essential curl

# Install Go 1.24.10
curl -OL https://go.dev/dl/go1.24.10.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.10.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Foundry (Forge, Cast, Anvil)
curl -L https://foundry.paradigm.xyz | bash
source ~/.bashrc
foundryup
```

---

## 2. Clone & Setup

```bash
git clone https://github.com/nowafinance/nowa-zk.git
cd nowa-zk
git submodule update --init --recursive
```

### clean old data (optional)

It will delete any old build data and ZK keys.

```bash
make clean-global
```

### Key Generation & Build

Run the full setup command to compile binaries and generate ZK keys:

```bash
make setup
```

This command will:
1.  Build the **Prover** and **Indexer** binaries.
2.  Compile the ZK circuit and run Trusted Setup.
3.  Generate artifacts in `~/.nowa-zk/keys/`.
4.  Generate and compile the `RollupVerifier.sol` contract.

---

## 3. Run Tests (Optional)

You can run the full test suite to ensure everything is working correctly:

```bash
make test
```

---

## 4. Local Run Guide

You will need **4 separate terminals** to run the full stack locally.

### Terminal 1: Start Local Blockchain (Optional)

If you don't have an external RPC, start a local anvil chain:

```bash
anvil
```

### Terminal 2: Deploy Contracts

Deploy the contracts to your local chain (or testnet).

```bash
# Export env vars from .env so forge can see them
set -a && source .env && set +a
cd contracts
mkdir deployments
forge script script/Deploy.s.sol --rpc-url $L1_RPC_URL --broadcast

# Copy deployment file to home directory so prover can auto-load it
cp deployments/deployments.json ~/.nowa-zk/deployments.json

cd ..
```

### Terminal 3: Start Indexer

```bash
# Run from project root (reads .env automatically)
make run-indexer
```

### Terminal 4: Start Prover

```bash
# Run from project root (reads .env automatically)
make run-prover
```
