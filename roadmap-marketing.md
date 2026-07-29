# 🗺️ Nowa-ZK: Engineering Roadmap

Welcome to the official engineering roadmap for **Nowa-ZK**, a cutting-edge protocol designed to execute high-frequency trades on a decentralized Cosmos network while settling immutably on Ethereum L1 via Zero-Knowledge Cryptography.

---

## 🚀 Vision & Architecture Snapshot

- **Mission:** Build a Tier-1 Omnichain ZK Rollup capable of processing thousands of trades per second with zero L1 gas fees.
- **Execution Layer:** A custom, decentralized Cosmos x/EVM blockchain.
- **Settlement Layer:** Ethereum L1 (Sepolia Testnet), secured by absolute mathematics.
- **Prover Engine:** High-performance Go-based ZK-SNARK Prover utilizing `gnark` (Groth16).

---

## 📊 Development Status Matrix

| Phase | Milestone | Status |
| :--- | :--- | :---: |
| **Phase 0 (v0.0.1)** | Core ZK Circuit & Decentralized Sequencer | ✅ **Complete** |
| **Phase 1 (v0.1.0)** | Ethereum L1 Settlement & Integration | ✅ **Complete** |
| **Phase 2 (v0.2.0)** | L1 Data Availability (DA) & ZK Scalability | ✅ **Complete** |
| **Phase 3 (v0.3.0)** | Universal Token State & Circuit Expansion | ⏳ **Next** |
| **Phase 4 (v0.4.0)** | The Escape Hatch & L1 Trustless Vaults | 📅 *Planned* |
| **Phase 5 (v0.5.0)** | The Multi-Chain ZK Bridge (Omnichain Hub) | 📅 *Planned* |
| **Phase 6 (v1.0.0)** | Decentralized Provers & Mainnet Level Codes Ready | 📅 *Planned* |

---

## 🏗️ Detailed Engineering Phases

### ✅ Phase 0: Core ZK Circuit & Sequencer (v0.0.1 - Completed)
*The foundation of the trading engine.*
* **SNARK Circuit:** Built the core Groth16 circuit in Go to validate EIP-712 signatures, MiMC hashes, and trade execution math.
* **Cosmos Indexer:** Designed a highly efficient daemon to sweep the execution layer, chunk transactions, and build fixed-size ZK batches.

### ✅ Phase 1: L1 Settlement & Verification (v0.1.0 - Completed)
*The cryptographic anchor to Ethereum.*
* **Smart Contracts:** Designed `TradeRegistry.sol` and generated the complex `TradeVerifier.sol` engine.
* **Prover Integration:** Connected the Go Prover directly to Ethereum RPCs to submit mathematically flawless `proof` and `publicInputs` payloads.

### ✅ Phase 2: L1 Data Availability (DA) & ZK Scalability (v0.2.0)
*Securing transaction history natively on Ethereum.*
* **Data Availability:** Upgrade `TradeRegistry.sol` to accept the raw `Trade[]` arrays as `calldata`.
* **Public Input Hashing:** Implement Fiat-Shamir hashing inside the ZK Circuit to compress thousands of trade inputs into a single manageable hash, completely bypassing the Ethereum EIP-170 size limits.

### 📅 Phase 3: Universal Token State & Circuit Expansion (v0.3.0)
*Ensuring Ethereum perfectly tracks all token movements, not just trades.*
* **Mint Circuits:** Write `mint_circuit.go` to prove authorized genesis token mints to Ethereum.
* **Transfer Circuits:** Write `transfer_circuit.go` to prove standard ERC20 peer-to-peer transfers.
* **L1 Sync:** Upgrade the Go Indexer to parse all Cosmos token events and mathematically sync the Cosmos State Root with the Ethereum State Root.

### 📅 Phase 4: The Escape Hatch (v0.4.0)
*Guaranteeing trustless self-custody and L1/L2 bridging.*
* **Trustless Withdrawals:** Build the `emergencyWithdraw()` function on L1, allowing users to rescue funds using standard Merkle Proofs if the Cosmos chain ever goes offline.
* **L1 Vaults:** Build the Ethereum Deposit Vaults that map 1:1 with L2 token balances.

### 📅 Phase 5: The Multi-Chain ZK Bridge (v0.5.0)
*Transforming the Rollup into an Omnichain Liquidity Hub via a Hub-and-Spoke architecture.*
* **The Ethereum Hub:** Establish Ethereum as the ultimate Source of Truth. The Go Prover submits all heavy ZK Proofs and Nullifiers exclusively to Ethereum to minimize gas costs and prevent multi-chain double spending.
* **"Dumb" Spoke Vaults:** Deploy lightweight, low-gas Deposit/Withdrawal Vaults to Binance Smart Chain, Arbitrum, Optimism, and Polygon.
* **Cross-Chain Messaging:** Connect the Ethereum Hub to the Spoke Vaults using a messaging protocol (e.g., LayerZero). Emergency withdrawals are verified purely on Ethereum, which then securely commands the Spoke Vaults to release funds.
* **Cross-Chain Indexing:** Connect the Cosmos Indexer to listen for Deposit events globally across all Spoke Vaults.

### 📅 Phase 6: Production & Decentralization (v1.0.0)
*Ready for Mainnet level deployment.*
* **Observability:** Prometheus metrics, Grafana dashboards, and structured logging.
* **DevOps:** Hardened Dockerfiles and GitHub Actions CI/CD pipelines.
* **Security:** Independent smart contract audits, circuit audits, and rate-limiting.
* **Decentralization:** Transitioning from a single centralized prover to a permissionless proving network where anyone can generate proofs for rewards.
