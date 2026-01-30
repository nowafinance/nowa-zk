# ZK-Sequencer - Nowa-ZK Network

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![Solidity](https://img.shields.io/badge/solidity-0.8.20-blue.svg)](https://soliditylang.org/)

This repository contains the official implementation of the **ZK-Sequencer** for the **nowa-zk** network.

---

## About nowa-zk

**nowa-zk** is an EVM-compatible, Cosmos-based blockchain. This sequencer is **not** an L2 for another network; it is a **core component of the Nowa-ZK L1 network itself**, designed to scale its own execution through zero-knowledge proofs.

### Architecture Overview

The ZK-Sequencer operates in a two-phase model:
- **Phase 1 (Current):** Centralized operation with a single sequencer and verifier operator
- **Phase 2 (Future):** Decentralized architecture with multiple operators

---

## 🔧 How It Works

The Nowa-ZK Sequencer is a critical infrastructure component with four primary responsibilities:

### 1. **Transaction Bundling**
- Accepts transactions from Nowa-ZK users
- Orders transactions deterministically
- Groups them into optimized batches

### 2. **State Execution**
- Executes batched transactions using EVM
- Computes state transitions using Sparse Merkle Trees (SMT)
- Validates account balances and nonces

### 3. **Proof Generation**
- Coordinates with the ZK Prover service
- Generates validity proofs using Groth16 (gnark)
- Proves correct state transitions

### 4. **On-Chain Verification**
- Submits ZK proofs to verifier contract
- Updates state root on-chain after finalization
- Implements challenge period for security

This architecture enables **high throughput** and **low-cost execution** while maintaining security through ZK proof verification.

---

## 🚀 Getting Started

We provide detailed setup guides for different environments:

| Environment | Description | Guide Link |
|-------------|-------------|------------|
| **Local Development** | Run everything on your local machine for testing and development. | **[📄 Local Setup Guide](docs/deployment/local.md)** |
| **Docker** | Run services in containerized environments. | **[🐳 Docker Setup Guide](docs/deployment/docker.md)** |
| **Cloud / Production** | Deploy to a production Linux server (Ubuntu). | **[☁️ Cloud Setup Guide](docs/deployment/cloud.md)** |

### Quick Pointers
- **Dependencies**: Go 1.21+, Foundry, Docker, Make
- **Key Artifacts**: Prover keys are generated in `make setup`
- **Configuration**: Uses `.env` for secrets and configuration

---

## 🏗️ Project Structure

```
nowa-zk/
├── contracts/          # Solidity smart contracts (Foundry)
├── sequencer/          # Go sequencer service
├── prover/             # Go ZK prover (Gnark)
├── docs/               # Documentation
├── scripts/            # Utility scripts
├── Dockerfile          # Multi-stage Docker build
├── docker-compose.yml  # Docker Compose orchestration
└── Makefile            # Project management commands
```

---

## 🛠️ Technology Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| **Smart Contracts** | Solidity | 0.8.20 |
| **Sequencer Backend** | Go | 1.21+ |
| **ZK Backend** | Gnark | Latest |
| **Proof System** | Groth16 | BN254 |
| **State Storage** | BadgerDB | v4 |

---

## � Features

### ✅ Implemented
- [x] Transaction batching & sequencing
- [x] State management (SMT, Accounts)
- [x] ZK Circuits (Groth16, MiMC)
- [x] Smart Contracts (BatchRegistry, StateManager)
- [x] Reorg protection & Graceful shutdown

### 🔜 Upcoming Features
- [ ] Gas fee calculation
- [ ] Decentralized sequencer network
- [ ] Performance optimization

---

## 📚 Documentation

- **[CODEME.md](CODEME.md)** - All CLI commands & scripts
- **[ROADMAP.md](ROADMAP.md)** - Development roadmap & milestones
- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Contribution guidelines
- **[SECURITY.md](SECURITY.md)** - Security policy & reporting

For API documentation, see **[docs/api.md](docs/api.md)**.

---

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

## 📜 License

This project is licensed under the **Apache License 2.0**. See [LICENSE](LICENSE) for details.

---

<div align="center">

**⚡ Fast • 🔒 Secure • 🌐 Decentralized**

</div>