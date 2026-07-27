<div align="center">
  <img src="assets/logo.png" alt="Nowa Logo" width="150" />
  <h1>Nowa-ZK: Indexer & Prover 🚀</h1>
  <p><b>⚡ Fast • 🔒 Secure • 🌐 Decentralized</b></p>

  [![License](https://img.shields.io/badge/License-BSL%201.1-blue.svg)](LICENSE)
  [![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()
  [![Go Version](https://img.shields.io/badge/go-1.24.10+-blue.svg)](https://golang.org/dl/)
  [![Solidity](https://img.shields.io/badge/solidity-0.8.20-blue.svg)](https://soliditylang.org/)
</div>

---

**Nowa-ZK** is a cutting-edge Layer 2 scaling solution leveraging Zero-Knowledge (ZK) proofs to ensure security, scalability, and fast finality. Built on a modular architecture involving Ethereum as Layer 1 and a Cosmos SDK-based EVM as Layer 2, Nowa-ZK offers a high-performance execution environment for decentralized applications.

This repository contains the official implementation of the **Indexer** (formerly ZK-Indexer), **Prover**, and **Smart Contracts** for the Nowa-ZK network.

---

## 🌟 Key Features

- **High-Performance Indexer:** Continuously indexes blocks from the decentralized Cosmos L2 Indexer and batches trades (e.g. 128 trades/batch) for efficient proof generation.
- **Succinct ZK Proofs:** Utilizes Groth16 to compress and verify EIP-712 trade signatures with minimal L1 footprint (~4KB per batch).
- **EVM Compatibility:** Built alongside a Cosmos EVM L2, allowing seamless deployment of Ethereum smart contracts.
- **Fast Finality:** Achieves ~1 second block times on L2 while maintaining Ethereum-grade security.

---

## 🏗 Architecture Overview

The system consists of three main operational components contained within this repository:

1. **[Indexer](./indexer)**: Acts as the data bridge coordinator. It indexes L2 data from the decentralized Cosmos Indexer, builds transaction batches, and provides data availability via a REST/WebSocket API.
2. **[Prover](./prover)**: The computational engine. It fetches batches from the Indexer, generates Groth16 Zero-Knowledge proofs, and submits them to L1.
3. **[Contracts](./contracts)**: The L1 foundation. Includes the `BatchRegistry` which verifies ZK proofs and manages the canonical sequence of verified trades on Ethereum.

For an in-depth architectural dive, please read our **[Litepaper](./litepaper.md)**.

---

## 🚀 Quick Start Guide

This guide provides immediate, functional commands to get the complete Nowa-ZK stack running locally using our `Makefile`. For a production or cloud server setup, please refer to the **[Cloud Deployment Guide](docs/deployment/cloud.md)**.

### Prerequisites
- **Go**: 1.24.10 or higher
- **Foundry**: Latest version (for compiling/deploying contracts)
- **Make**: For running automated setup commands
- **Git**: To clone the repository

### 1. Clone & Initialize

```bash
git clone https://github.com/nowafinance/nowa-zk.git
cd nowa-zk
```

### 2. Full Setup

The `setup` command compiles the binaries, runs the Trusted Setup for ZK keys, and generates the `RollupVerifier.sol` contract. All generated files are safely stored in `~/.nowa-zk/`.

<!-- Wipe everything (Data, Keys, and Global Configs) -->
<!-- make clean-global -->

```bash
make setup
```

### 3. Deploy Smart Contracts

Create and edit a `.env` file in the root directory to configure the network connections and provide your deployment private key:

```bash
nano .env
```

Add the following required keys into the `.env` file:
```env
L2_RPC_URL=https://archival-node.nowa.finance
L1_RPC_URL=https://ethereum-sepolia-rpc.publicnode.com
PRIVATE_KEY=0xYOUR_PRIVATE_KEY_HERE
```

Save the file, then deploy the contracts:

```bash
make deploy
```
*Note: This command automatically saves your contract deployment addresses to `~/.nowa-zk/deployments.json` so the Prover can find them.*

### 4. Start the System

The `Makefile` will automatically read your `.env` variables. Open two separate terminals and start the Indexer and Prover:

**Terminal 1 (Start Indexer):**
```bash
make run-indexer
```

**Terminal 2 (Start Prover):**
```bash
make run-prover
```

---

## 📚 Documentation & Resources

Dive deeper into the Nowa-ZK ecosystem:

- 📄 **[Litepaper](litepaper.md)** - Full architectural and technical overview.
- 🗺️ **[Roadmap](ROADMAP.md)** - Development milestones and future phases.
- 🔌 **[API Documentation](docs/api.md)** - Detailed endpoints for Indexer and Prover interaction.
- 🛡️ **[Security Policy](SECURITY.md)** - Vulnerability reporting and security guidelines.

**Component READMEs:**
- [Indexer Documentation](indexer/README.md)
- [Prover Documentation](prover/README.md)
- [Smart Contracts Documentation](contracts/README.md)

---

## 🤝 Contributing

We welcome and appreciate contributions from the community! Whether it's bug reports, feature requests, or code contributions, please check out our **[Contribution Guidelines](CONTRIBUTING.md)** before getting started.

Please adhere to our **[Code of Conduct](CODE_OF_CONDUCT.md)** when interacting within our community.

---

## 📜 License

This project is licensed under the **Business Source License 1.1**. See the [LICENSE](LICENSE) file for full details.