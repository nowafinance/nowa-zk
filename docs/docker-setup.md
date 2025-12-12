# 🐳 Docker Setup Guide

This guide describes how to run the **Tan-ZK** system (Sequencer & Prover) using Docker, connecting to an **external blockchain network** (e.g., Cloud Testnet, Sepolia, or a local node running on host).

## Prerequisites

*   **Docker** installed (includes `docker compose`).
*   **Go** & **Make** (for generating keys and deploying contracts on the host).
*   **Foundry** (for `cast` and `forge`).

## Configuration

The system uses the `.env` file for configuration. Ensure your `.env` file is set up correctly in the project root.

**Required `.env` Variables:**
```bash
# RPC URL of the Layer 1 Blockchain
# If connecting to a node on the host machine, use http://host.docker.internal:8545
RPC=http://host.docker.internal:8545

# Private Key for transactions (Sequencer/Prover address)
PRIVATE_KEY=your_private_key_here

# start indexing from block
INDEX_FROM_BLOCK=0
```

## Quick Start

### 1. Setup & Key Generation (Host)

Generate cryptographic keys and compile contracts locally.

```bash
# Clean previous artifacts
make clean

# Install dependencies
make deps

# Generate Zero-Knowledge Keys (Stored in .tan-zk/keys)
make setup
```

### 2. Deploy Contracts (Host)

Deploy the smart contracts to your target network.

```bash
# Ensure your .env has the correct RPC and PRIVATE_KEY before running this!
make deploy
```
> **Note:** This command updates `.tan-zk/deployments.json` and `.tan-zk/secrets.env`.

### 3. Start Services

Start the Sequencer and Prover containers. They will automatically load the `RPC` and `PRIVATE_KEY` from your `.env` file.

```bash
docker compose up -d
```

---

## Verifying the System

### Check Logs
View the logs for each service to ensure they are connected and running:

```bash
# Follow sequencer logs
docker compose logs -f sequencer

# Follow prover logs
docker compose logs -f prover
```

### Interact with API
The Sequencer API is exposed on port **8080**.

```bash
# Check Status
curl http://localhost:8080/status

# Check Latest Batch
curl http://localhost:8080/batch/latest
```

---

## Troubleshooting

### "Connection refused"
*   Ensure your RPC node is running and accessible.
*   If using a local node on the host, ensure you use `http://host.docker.internal:PORT` instead of `localhost`.

### "Contract address required"
*   Ensure `make deploy` ran successfully.
*   Check that `.tan-zk/deployments.json` exists and contains the correct address.

### Resetting Data
To wipe the container data (database) and start fresh:

```bash
docker compose down -v
rm -rf dir_data/
```
