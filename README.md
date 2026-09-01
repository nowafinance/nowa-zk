<div align="center">
  <img src="assets/logo.png" alt="Nowa Logo" width="150" />
  <h1>Nowa-ZK 🚀</h1>
  <p><b>⚡ Fast • 🔒 Secure • 🌐 ZK-Verified</b></p>

  [![License](https://img.shields.io/badge/License-BSL%201.1-blue.svg)](LICENSE)
  [![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()
  [![Go Version](https://img.shields.io/badge/go-1.25.7+-blue.svg)](https://golang.org/dl/)
  [![Solidity](https://img.shields.io/badge/solidity-0.8.30-blue.svg)](https://soliditylang.org/)
</div>

---

**Nowa-ZK** is an app-specific, ZK-verified orderbook DEX. Trades are matched and
executed on a dedicated execution layer, then settled with a cryptographic proof
anchored on Ethereum — combining fast, low-cost trading with verifiable trade
integrity.

## How It Works

1. **Trade** — orders are matched and executed on Nowa's dedicated execution layer,
   built for orderbook speed rather than general-purpose smart contracts.
2. **Prove** — batches of trades are compressed into a succinct zk-SNARK (Groth16)
   proof, cryptographically guaranteeing every trade in the batch was valid.
3. **Settle** — the proof is anchored on Ethereum L1, permanently fixing that batch's
   trades against tampering.

This repository contains the execution, proving, and settlement components that make
up that pipeline. Specifics of which components are active in which environment are
subject to change and aren't detailed here — see the team for current status.

---

## 🌟 Capabilities

- **ZK-Verified Trade Settlement** — every batch is proven with a Groth16 zk-SNARK and
  anchored on Ethereum.
- **Wallet-Native Trading** — orders can be signed with a standard Ethereum wallet.
- **Dedicated Execution Speed** — a purpose-built execution layer for orderbook
  throughput.
- **On-Chain Data Availability** — batch data is posted to Ethereum, not withheld
  off-chain.
- **L1 ↔ L2 Deposits & Withdrawals** — asset bridging between Ethereum and the
  execution layer.

---

## 🚀 Quick Start Guide

### Prerequisites
- **OS**: Ubuntu 22.04 LTS (or compatible Linux)
- **Hardware**: 16GB+ RAM, 4+ CPU cores, 50GB+ SSD (the Prover is compute-intensive)
- **Go**: 1.25.7 or higher
- **Foundry**: latest (Forge, Cast, Anvil)
- **Make**, **Git**, **jq**, **python3**

### 1. Clone & Configure

```bash
git clone https://github.com/nowafinance/nowa-zk.git
cd nowa-zk
cp .env.example .env
```
Edit `.env` with your own RPC URL and key before building or deploying anything.

Build, setup, and deploy commands are defined in the `Makefile` — run `make help` (or
open the `Makefile` directly) for the current list of available targets.

---

## 🤝 Contributing

We welcome and appreciate contributions from the community! Whether it's bug reports, feature requests, or code contributions, please check out our **[Contribution Guidelines](CONTRIBUTING.md)** before getting started.

Please adhere to our **[Code of Conduct](CODE_OF_CONDUCT.md)** when interacting within our community.

---

## 📜 License

This project is licensed under the **Business Source License 1.1**. See the [LICENSE](LICENSE) file for full details.
