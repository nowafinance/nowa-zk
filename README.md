<div align="center">
  <img src="assets/logo.png" alt="Nowa Logo" width="150" />
  <h1>Nowa-ZK Sequencer 🚀</h1>
  <p><b>⚡ Fast • 🔒 Secure • 🌐 Decentralized</b></p>

  [![License](https://img.shields.io/badge/License-BSL%201.1-blue.svg)](LICENSE)
  [![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()
  [![Go Version](https://img.shields.io/badge/go-1.24.10+-blue.svg)](https://golang.org/dl/)
  [![Solidity](https://img.shields.io/badge/solidity-0.8.20-blue.svg)](https://soliditylang.org/)
</div>

---

**Nowa-ZK** is a cutting-edge Layer 2 scaling solution leveraging Zero-Knowledge (ZK) proofs to ensure security, scalability, and fast finality. Built on a modular architecture involving Ethereum as Layer 1 and a Cosmos SDK-based EVM as Layer 2, Nowa-ZK offers a high-performance execution environment for decentralized applications.

This repository contains the official implementation of the **ZK-Sequencer**, **Prover**, and **Smart Contracts** for the Nowa-ZK network.

---

## 🌟 Key Features

- **High-Performance Sequencer:** Continuously indexes blocks and batches transactions (128 txs/batch) for efficient proof generation.
- **Succinct ZK Proofs:** Utilizes Groth16 to compress transactions and prove state transitions with minimal L1 footprint (~4KB per batch).
- **EVM Compatibility:** Built alongside a Cosmos EVM L2, allowing seamless deployment of Ethereum smart contracts.
- **Fast Finality:** Achieves ~1 second block times on L2 while maintaining Ethereum-grade security.

---

## 🏗 Architecture Overview

The system consists of three main operational components contained within this repository:

1. **[Sequencer](./sequencer)**: Acts as the coordinator. It indexes L2 data, builds transaction batches, and provides data availability via a REST/WebSocket API.
2. **[Prover](./prover)**: The computational engine. It fetches batches from the Sequencer, generates Groth16 Zero-Knowledge proofs, and submits them to L1.
3. **[Contracts](./contracts)**: The L1 foundation. Includes the `BatchRegistry` which verifies ZK proofs and manages the canonical L2 state on Ethereum.

For an in-depth architectural dive, please read our **[Litepaper](./litepaper.md)**.

---

## 🚀 Quick Start Guide

This guide provides immediate, functional commands to get the complete Nowa-ZK stack running locally. For a production or cloud server setup, please refer to the **[Cloud Deployment Guide](docs/deployment/cloud.md)**.

### Prerequisites
- **Go**: 1.24.10 or higher
- **Foundry**: Latest version (for compiling/deploying contracts)
- **Git**: To clone the repository

### 1. Clone & Initialize

```bash
git clone https://github.com/nowafinance/nowa-zk.git
cd nowa-zk
```

### 2. Clean Up (Optional)
*Run this if you have previously started the system and need a fresh state.*

```bash
# Clear Sequencer Data
rm -rf sequencer/data/

# Clean Prover Keys & Contracts
rm -rf prover/keys/

# Clean compiled contracts
(cd contracts && forge clean)
```

### 3. Build & Setup Components

**Generate Prover Keys & Verifier Contract:**
```bash
cd prover
go run ./cmd/prover setup
cd ..
```

**Build Smart Contracts:**
```bash
cd contracts
forge build
cd ..
```

### 4. Deploy Smart Contracts

Create a `.env` file in the `contracts` directory with your `RPC_URL` and `PRIVATE_KEY`.

```bash
cd contracts

export RPC_PROVER=https://archival-node.nowa.finance
# RPC_PROVER=https://ethereum-sepolia-rpc.publicnode.com
export PRIVATE_KEY=0x.........

forge script script/Deploy.s.sol:Deploy --rpc-url $RPC_PROVER --private-key $PRIVATE_KEY --broadcast

cd ..
```

### 5. Start the System

**Start the Sequencer:**
```bash
cd sequencer
export RPC_URL=https://archival-node.nowa.finance  # Point to your L2 node/RPC
go run ./cmd/sequencer start
```

**Start the Prover** *(in a new terminal)*:
```bash
cd prover
export DEPLOYED_CONTRACT_ADDRESS=<DEPLOYED_CONTRACT_ADDRESS> # deployments.json -> BatchRegistry  
export PRIVATE_KEY=<YOUR_PRIVATE_KEY>
export RPC_PROVER=https://node1.nowa.finance

go run ./cmd/prover start --contract $DEPLOYED_CONTRACT_ADDRESS --private-key $PRIVATE_KEY
```

---

## 📚 Documentation & Resources

Dive deeper into the Nowa-ZK ecosystem:

- 📄 **[Litepaper](litepaper.md)** - Full architectural and technical overview.
- 🗺️ **[Roadmap](ROADMAP.md)** - Development milestones and future phases.
- 🔌 **[API Documentation](docs/api.md)** - Detailed endpoints for Sequencer and Prover interaction.
- 🛡️ **[Security Policy](SECURITY.md)** - Vulnerability reporting and security guidelines.

**Component READMEs:**
- [Sequencer Documentation](sequencer/README.md)
- [Prover Documentation](prover/README.md)
- [Smart Contracts Documentation](contracts/README.md)

---

## 🤝 Contributing

We welcome and appreciate contributions from the community! Whether it's bug reports, feature requests, or code contributions, please check out our **[Contribution Guidelines](CONTRIBUTING.md)** before getting started.

Please adhere to our **[Code of Conduct](CODE_OF_CONDUCT.md)** when interacting within our community.

---

## 📜 License

This project is licensed under the **Business Source License 1.1**. See the [LICENSE](LICENSE) file for full details.