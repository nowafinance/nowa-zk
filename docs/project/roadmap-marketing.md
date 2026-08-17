# 🗺️ Nowa-ZK: Engineering Roadmap

Welcome to the official engineering roadmap for **Nowa-ZK**, a cutting-edge ZK-Rollup Orderbook DEX designed to execute high-frequency trades off-chain while settling immutably on Layer 1 via Zero-Knowledge Cryptography.

---

## 🚀 Vision & Architecture Snapshot

- **Mission:** Build a Tier-1 ZK Rollup capable of processing thousands of trades per second with zero L1 gas fees.
- **Execution Layer:** A custom, ultra-fast off-chain Matching Engine / Sequencer (No blockchain consensus bottleneck on L2).
- **Settlement Layer:** Ethereum L1 (Sepolia Testnet), secured by absolute mathematics and ZK-SNARKs.
- **Prover Engine:** High-performance Go-based ZK-SNARK Prover utilizing `gnark` (Groth16).

---

> [!NOTE]
> This tracks phases going forward. For what's actually been **published as a release**
> vs. built-but-unreleased vs. genuinely incomplete right now, see
> [Release Status](./release-status.md).

## 📊 Development Status Matrix

| Phase | Milestone | Status |
| :--- | :--- | :---: |
| **Phase 0 (v0.0.1)** | Core ZK Circuit & Stateless Verification | ✅ **Complete** |
| **Phase 1 (v0.1.0)** | Ethereum L1 Settlement & Integration | ✅ **Complete** |
| **Phase 2 (v0.2.0)** | L1 Data Availability (DA) & ZK Scalability | ✅ **Complete** |
| **Phase 3 (v0.3.0)** | Universal State Transition (The Rollup Engine) | ✅ **Complete** |
| **Phase 4 (v0.4.0)** | The High-Frequency Go Sequencer | 🟡 **Mostly Complete** — matching engine, persistent LevelDB SMT, REST API, and ZK Batcher are built and live; WebSocket streaming is still outstanding |
| **Phase 5 (v0.5.0)** | The L1 Bridge & Escape Hatch | 🟡 **Partially Complete** — L1 deposits + a live deposit watcher are built; the Escape Hatch (`emergencyWithdraw()`) is **not implemented** — `NowaRollup.withdraw()` is currently an owner-only placeholder |
| **Phase 6 (v1.0.0)** | Security Audits & Mainnet Beta | 📅 *Planned* |
| **Phase 7 (v2.0.0)** | Decentralized Provers & Ecosystem Expansion | 📅 *Planned* |

*(Status last verified against code 2026-08-17. See [architecture/overview.md](../architecture/overview.md#known-gaps-as-of-2026-08-17) for the current gap list.)*

---

## 🏗️ Detailed Engineering Phases

### ✅ Phase 0 & 1: Core ZK Circuit & L1 Settlement (Completed)
*The foundation of cryptographic verification.*
* **SNARK Circuit:** Built the core Groth16 circuit in Go, now verifying EdDSA (BabyJubJub) trade/transfer/withdrawal/deposit signatures — see Phase 4 for why EdDSA replaced the originally-planned EIP-712/ECDSA scheme.
* **Smart Contracts:** `NowaRollup.sol` (renamed from `TradeRegistry.sol`) + a generated `Verifier.sol` mathematically settle proofs on Ethereum.

### ✅ Phase 2: L1 Data Availability (DA) (Completed)
*Securing transaction history natively on Ethereum.*
* **Data Availability:** Upgraded contracts to accept raw operation payloads, ensuring the raw transaction history is permanently stored on Ethereum (preventing Validium lockups).

### ✅ Phase 3: Universal State Transition Circuit (Completed)
*Transforming from a simple verifier into a stateful ZK-Rollup.*
* **Sparse Merkle Trees:** Implemented Depth-20 SMTs inside the circuit to mathematically track user balances and nonces.
* **Unified Operations:** A single circuit that seamlessly processes Deposits, Trades, Transfers, and Withdrawal Requests while strictly enforcing balance constraints.

### 🟡 Phase 4: The High-Frequency Go Sequencer (v0.4.0) — Mostly Complete
*Replacing slow blockchain consensus with a lightning-fast matching engine.*
* ✅ **The Trading Engine:** A centralized, high-frequency orderbook matching engine in Go (`sequencer/internal/engine`), live.
* ✅ **Persistent Merkle DB:** The Sparse Merkle Tree is persisted in LevelDB (`sequencer/internal/state`), depth 28.
* 🔲 **High-Frequency API:** REST is live (`/order`, `/orderbook`, `/balance`, `/account`, `/batch/*`); **WebSocket streaming is not yet built.**
* ✅ **The ZK Batcher:** Matched trades are batched (`sequencer/internal/batcher`) and handed to the Prover — current `BatchSize = 1` (one fill per proof), not the fixed multi-trade batches originally envisioned; see the recursive-proving discussion in `FAQ-ZK.md` for the scaling plan.

### 🟡 Phase 5: The L1 Bridge & Escape Hatch (v0.5.0) — Partially Complete
*Connecting the Rollup to real Ethereum value while guaranteeing self-custody.*
* ✅ **L1 Deposit Vaults:** `NowaRollup.deposit()` locks ERC20 tokens; a live Deposit Watcher (`sequencer/cmd/sequencer/deposit_watcher.go`) mints the corresponding L2 balance automatically.
* ❌ **Trustless Withdrawals (The Escape Hatch):** **Not implemented.** `NowaRollup.withdraw()` today is an explicit placeholder — `onlyOwner`, and it ignores the Merkle proof parameter entirely. There is currently no `emergencyWithdraw()` and no way for a user to self-rescue funds if the Sequencer goes offline. This is the top-priority remaining item in this phase.

### 📅 Phase 6: Security Audits & Mainnet Beta (v1.0.0)
*Hardening the protocol for production.*
* **Full System Audits:** Formal verification of the Gnark circuits and third-party security audits for the L1 bridge contracts.
* **Mainnet Launch:** Deploy to Ethereum Mainnet with guarded deposit limits and strict API rate limiting.

### 🔮 Phase 7: Decentralized Provers & Ecosystem Expansion (v2.0.0)
*The endgame of decentralization.*
* **Prover Market:** Allow anyone to run a Prover node and compete to generate proofs for the Sequencer in exchange for protocol rewards.
* **Nowa Chain:** Expand the ZK-Rollup into a fully Sovereign L1 or L3 AppChain ecosystem.
