# ZK-Sequencer - ⚠️ Under Production

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/tannetwork/zk-sequencer/main.yml?branch=main)](https://github.com/tannetwork/zk-sequencer/actions)

This repository contains the official implementation of the **ZK-Sequencer** for the **Tan-ZK** network.

**Tan-ZK** is an EVM-compatible, Cosmos-based blockchain. This sequencer is **not** an L2 for another network; it is a **core component of the `Tan-ZK` L1 network itself**, designed to scale its own execution.   

## About This Sequencer

The Tan-ZK Sequencer is a critical part of the network's infrastructure. In phase 1 it operates in a centralized mode (single operator for sequencer and verifier); future phases will decentralize both components. Its primary responsibilities are:

* **Transaction Bundling:** Accepting transactions from `Tan-ZK` users, ordering them, and forming them into batches.
* **State Execution:** Executing the batched transactions and computing the new state root.
* **Proof Generation:** Coordinating with the ZK Prover to generate a ZK validity proof for the entire batch of transactions.
* **On-Chain Verification:** Submitting this ZK proof to a verifier smart contract deployed on the **main `Tan-ZK` blockchain** for finalization and settlement.

This architecture allows `Tan-ZK` to achieve high throughput and low-cost execution while anchoring its security and final state to its own consensus layer.

## Technology Stack

This project is built using a dedicated stack to optimize for performance and security:
* **Smart Contracts (Verifier):** Solidity (Foundry)
* **Sequencer Service:** Go 1.21+
* **ZK Prover:** Go (Gnark)

## Project Status

This project is in active development. For a detailed breakdown of our development plan, milestones, and release targets, please see **[ROADMAP.md](ROADMAP.md)**.

## Getting Started

Quick start for local development (contracts + services):

### Prerequisites
* Go 1.21+
* Foundry
* Docker (optional, for infra/monitoring)

### Build & Test
```bash
# Clone the repository
git clone https://github.com/tannetwork/zk-sequencer.git
cd zk-sequencer

# Contracts (Foundry)
cd contracts
forge build
forge test

# Prover (Go + Gnark)
cd ../prover
go build ./...
go test ./...

# Sequencer (Go)
cd ../sequencer
go build ./...
```

## Contributing

We welcome contributions from the community\! If you're interested in helping build the future of Tan-ZK, please read our [CONTRIBUTING.md](CONTRIBUTING.md) (coming soon) for guidelines.

## License

This project is licensed under the **Apache License 2.0**. See the [LICENSE](LICENSE) file for details.
