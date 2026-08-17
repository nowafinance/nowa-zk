<div align="center">
  <img src="assets/logo.png" alt="Nowa Logo" width="150" />
  <h1>Nowa-ZK: Sequencer, Prover & Contracts 🚀</h1>
  <p><b>⚡ Fast • 🔒 Secure • 🌐 ZK-Verified</b></p>

  [![License](https://img.shields.io/badge/License-BSL%201.1-blue.svg)](LICENSE)
  [![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)]()
  [![Go Version](https://img.shields.io/badge/go-1.25.7+-blue.svg)](https://golang.org/dl/)
  [![Solidity](https://img.shields.io/badge/solidity-0.8.30-blue.svg)](https://soliditylang.org/)
  [![Release](https://img.shields.io/badge/release-v0.4.0-blue.svg)](https://github.com/nowafinance/nowa-zk/releases/tag/v0.4.0)
</div>

---

**Nowa-ZK** is an app-specific ZK-rollup orderbook DEX. Trades are matched off-chain by
a high-frequency Go **Sequencer**, and every state transition is proven correct with a
Groth16 zk-SNARK and settled on Ethereum (Sepolia) via a **Prover** and a set of L1
**Contracts**.

> [!NOTE]
> **Project status**: core matching, proving, and L1 settlement work end-to-end today —
> including EIP-4844 blob data availability, verified live on Sepolia. Self-custody is
> **not** trustless yet: `withdraw()` is currently owner-gated with no escape hatch. See
> **[Release Status](docs/project/release-status.md)** for the full completed-vs-incomplete
> breakdown before treating any deployment as holding real funds.

---

## 🌟 Key Features

- **Off-Chain Matching Engine:** In-memory limit-order orderbook (price/time priority,
  partial fills) backed by a persistent Sparse Merkle Tree (depth 28) in LevelDB.
- **Universal State Transition Circuit:** One Groth16 circuit verifies Trades,
  Transfers, Withdrawals, and Deposits via Merkle inclusion proofs — not just signatures.
- **Real L1 Data Availability:** Full batch transition data posted to Ethereum via
  EIP-4844 blobs (post-Osaka cell-proof format), not withheld off-chain.
- **Automatic L1 → L2 Deposits:** A live Deposit Watcher mints L2 balance the moment a
  `deposit()` lands on L1 — no manual relay.
- **Dual-Token Swaps:** Trades aren't limited to one fixed pair.

---

## 🏗 Architecture Overview

Three components make up the live trading pipeline:

1. **[Sequencer](./sequencer)**: The execution layer. Matches orders, maintains the L2
   account tree, and seals batches for the Prover via a REST API (`:8080`).
2. **[Prover](./prover)**: Fetches sealed batches from the Sequencer, generates Groth16
   proofs, packs an EIP-4844 DA blob, and submits both to L1.
3. **[Contracts](./contracts)**: `NowaRollup.sol` verifies proofs (via the generated
   `Verifier.sol`), enforces DA blob presence, and tracks `stateRoot`/`batchCount`.

An earlier design ran execution on a Cosmos-SDK chain with an **[Indexer](./indexer)**
polling blocks — that model has been replaced by the Sequencer above. The Indexer still
builds and runs (`make run-indexer`) but is legacy/optional, off the live path.

For the full picture — component responsibilities, data flow, and honestly-documented
known gaps — see **[docs/architecture/overview.md](docs/architecture/overview.md)**.

---

## 🚀 Quick Start Guide

Functional commands to run the full stack locally via the `Makefile`. For production /
cloud setup, see the **[Local](docs/deployment/local.md)** or
**[Cloud Deployment](docs/deployment/cloud.md)** guides — both include the complete
first-run *and* full-reset command sequences.

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
Edit `.env` — at minimum, set `L1_RPC_URL` and a funded `PRIVATE_KEY`.

### 2. Full Setup

Compiles the circuit, runs the Groth16 trusted setup, and generates
`contracts/src/generated/Verifier.sol`. Generated keys are stored in `~/.nowa-zk/`.

```bash
make setup
make build
```

### 3. Deploy Contracts

```bash
set -a; source .env; set +a
make deploy
```
Saves deployment addresses to `~/.nowa-zk/deployments.json`, which the Sequencer and
Prover both auto-load from.

> [!IMPORTANT]
> A fresh `NowaRollup` starts at `stateRoot = 0`, but no real Sequencer tree — empty or
> not — ever actually roots to `0`. Skipping the one-time bootstrap makes every
> `submitBatch()` revert, spending real gas each time. See
> [docs/deployment/local.md §4](docs/deployment/local.md#4-first-time-run) for the exact
> fix — it's a single `cast send setStateRoot(...)` call, done once.

### 4. Start the System

Two terminals:

**Terminal 1 (Sequencer):**
```bash
make run-sequencer
```

**Terminal 2 (Prover):**
```bash
make run-prover
```

Then place a real matched trade to see it flow through:
```bash
cd sequencer && go run ./cmd/cli/test_client.go
curl http://localhost:8080/batch/latest
```

---

## 📚 Documentation & Resources

- 📊 **[Release Status](docs/project/release-status.md)** — what's released, what's shipped-but-unreleased, and what's genuinely incomplete.
- 📐 **[Architecture Overview](docs/architecture/overview.md)** — components, data flow, storage, and known gaps.
- 🧪 **[Testing Guide](docs/testing.md)** — unit tests per component plus a live end-to-end drill.
- ❓ **[Architecture FAQ](FAQ-ZK.md)** — ZK-Rollup vs. Validium, DA, public input hashing, why Groth16, why EdDSA.
- 🗺️ **[Roadmap](docs/project/roadmap-marketing.md)** — development phases and milestone status.
- 🛡️ **[Security Policy](SECURITY.md)** — vulnerability reporting and security guidelines.

> [!NOTE]
> `litepaper.md` describes an earlier Cosmos-execution-layer design and hasn't been
> updated for the current Sequencer-based architecture — treat the docs above as the
> current source of truth in the meantime.

**Component READMEs:**
- [Prover Documentation](prover/README.md)
- [Smart Contracts Documentation](contracts/README.md)
- [Indexer Documentation](indexer/README.md) *(legacy/optional component)*

---

## 🤝 Contributing

We welcome and appreciate contributions from the community! Whether it's bug reports, feature requests, or code contributions, please check out our **[Contribution Guidelines](CONTRIBUTING.md)** before getting started.

Please adhere to our **[Code of Conduct](CODE_OF_CONDUCT.md)** when interacting within our community.

---

## 📜 License

This project is licensed under the **Business Source License 1.1**. See the [LICENSE](LICENSE) file for full details.
