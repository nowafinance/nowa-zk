# ZK-Sequencer - Nowa-ZK Network

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![Solidity](https://img.shields.io/badge/solidity-0.8.20-blue.svg)](https://soliditylang.org/)

This repository contains the official implementation of the **ZK-Sequencer** for the **nowa-zk** network.

---

## 🚀 Getting Started

We provide detailed setup guides for different environments, but here is a quick start guide to run the system:

### 1. Build Required Files

#### Generate Prover Keys and Verifier Contract

```bash
cd prover
go run ./cmd/prover setup
```

#### Build Contracts

```bash
cd ../contracts
forge build
```

### 2. Configure Environment & Deploy

Create a `.env` file in the `contracts` directory with your `RPC` and `PRIVATE_KEY`, then deploy:

```bash
cd contracts
set -a
source .env
set +a
forge script script/Deploy.s.sol:Deploy --rpc-url $RPC --private-key $PRIVATE_KEY --broadcast
```

### 3. Start the System

**Start Sequencer:**
```bash
cd sequencer
export RPC=https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY
go run ./cmd/sequencer start
```

**Start Prover:**
```bash
cd prover
go run ./cmd/prover start --contract 0xYOUR_CONTRACT_ADDRESS --private-key 0xYOUR_PRIVATE_KEY
```

---

## 📚 Documentation

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