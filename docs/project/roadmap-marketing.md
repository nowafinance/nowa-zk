# 🗺️ Nowa-ZK: Engineering Roadmap

Welcome to the official engineering roadmap for **Nowa-ZK**, a cutting-edge ZK-Rollup Orderbook DEX designed to execute high-frequency trades off-chain while settling immutably on Layer 1 via Zero-Knowledge Cryptography.

---

## 🚀 Vision & Architecture Snapshot

- **Mission:** Build a Tier-1 ZK Rollup capable of processing thousands of trades per second with zero L1 gas fees.
- **Execution Layer:** A custom, ultra-fast off-chain Matching Engine / Sequencer (No blockchain consensus bottleneck on L2).
- **Settlement Layer:** Ethereum L1 (Sepolia Testnet), secured by absolute mathematics and ZK-SNARKs.
- **Prover Engine:** High-performance Go-based ZK-SNARK Prover utilizing `gnark` (Groth16).

---

## 📊 Development Status Matrix

| Phase | Milestone | Status |
| :--- | :--- | :---: |
| **Phase 0 (v0.0.1)** | Core ZK Circuit & Stateless Verification | ✅ **Complete** |
| **Phase 1 (v0.1.0)** | Ethereum L1 Settlement & Integration | ✅ **Complete** |
| **Phase 2 (v0.2.0)** | L1 Data Availability (DA) & ZK Scalability | ✅ **Complete** |
| **Phase 3 (v0.3.0)** | Universal State Transition (The Rollup Engine) | ✅ **Complete** |
| **Phase 4 (v0.4.0)** | Off-Chain Sequencer & Orderbook Core | ⏳ **Next** |
| **Phase 5 (v0.5.0)** | Persistent Merkle DB & Crash Recovery | 📅 *Planned* |
| **Phase 6 (v0.6.0)** | High-Frequency API & WebSocket Feeds | 📅 *Planned* |
| **Phase 7 (v0.7.0)** | Internal Devnet & Shadow Trading | 📅 *Planned* |
| **Phase 8 (v0.8.0)** | The Escape Hatch (Trustless Withdrawals) | 📅 *Planned* |
| **Phase 9 (v0.9.0)** | Full System Audits & Rate Limiting | 📅 *Planned* |
| **Phase 10 (v1.0.0)** | **Real L1/L2 Bridge & Mainnet Beta** | 📅 *Planned* |
| **Phase 11 (v2.0.0)** | Decentralized Prover Network | 📅 *Planned* |
| **Phase 12 (v3.0.0)** | The Year 2 Crossroads (Ultimate App vs Ecosystem Hub) | 📅 *Planned* |

---

## 🏗️ Detailed Engineering Phases

### ✅ Phase 0 & 1: Core ZK Circuit & L1 Settlement (Completed)
*The foundation of cryptographic verification.*
* **SNARK Circuit:** Built the core Groth16 circuit in Go to validate EIP-712 signatures.
* **Smart Contracts:** Designed `TradeRegistry.sol` and generated the complex `Verifier.sol` engine to mathematically settle proofs on Ethereum.

### ✅ Phase 2: L1 Data Availability (DA) (Completed)
*Securing transaction history natively on Ethereum.*
* **Data Availability:** Upgraded contracts to accept raw operation payloads, ensuring the raw transaction history is permanently stored on Ethereum (preventing Validium lockups).

### ✅ Phase 3: Universal State Transition Circuit (Completed)
*Transforming from a simple verifier into a stateful ZK-Rollup.*
* **Sparse Merkle Trees:** Implemented Depth-20 SMTs inside the circuit to mathematically track user balances and nonces.
* **Unified Operations:** A single circuit that seamlessly processes Deposits, Trades, Transfers, and Withdrawal Requests while strictly enforcing balance constraints.

### 📅 Phase 4: Off-Chain Sequencer & Orderbook Core (v0.4.0)
*Replacing slow blockchain consensus with a lightning-fast matching engine.*
* **The Trading Engine:** Build a centralized, high-frequency orderbook matching engine in Go. 
* **Gasless Signatures (BabyJubJub):** Transition users from paying L2 gas fees to signing free, off-chain messages. Users will generate a SNARK-friendly EdDSA (BabyJubJub) "Trading Key" in their browser for maximum ZK proving efficiency.
* **The Sequencer:** The engine verifies these EdDSA signatures instantly in Go, matches trades, and chunks transactions into fixed-size ZK batches.
* **No L2 Blockchain:** Remove all dependencies on an L2 blockchain (like Cosmos/EVM). The engine *is* the execution layer, allowing NASDAQ-level trading speeds.

### 📅 Phase 5: Persistent Merkle DB & Crash Recovery (v0.5.0)
*Ensuring the Sequencer never loses state.*
* **Database Integration:** Migrate the in-memory Sparse Merkle Tree and Orderbook to a persistent database (LevelDB or a Kafka stream, depending on performance testing).
* **State Recovery:** Build tooling to perfectly reconstruct the Sequencer's database purely from Ethereum L1 calldata (The ultimate disaster recovery).

### 📅 Phase 6: High-Frequency API & WebSockets (v0.6.0)
*Opening the DEX to the world.*
* **REST API:** Endpoints for users to query their L2 balances and order history.
* **WebSockets:** Real-time orderbook feeds and execution streams for algorithmic traders.

### 📅 Phase 7: Internal Devnet & Shadow Trading (v0.7.0)
*Stress testing the engine.*
* **Mock Traffic:** Deploy bots that execute 1,000 TPS to ensure the Prover can keep up with the Sequencer.
* **Gas Optimization:** Tune the batch sizes and Ethereum submission intervals for maximum cost efficiency.

### 📅 Phase 8: The Escape Hatch (v0.8.0)
*Guaranteeing trustless self-custody.*
* **Trustless Withdrawals:** Build the `emergencyWithdraw()` function on L1, allowing users to rescue funds using standard Merkle Proofs if the off-chain Sequencer ever goes offline.

### 📅 Phase 9: Full System Audits (v0.9.0)
*Security first.*
* **Circuit Audit:** Formal verification of the Gnark circuits to ensure no fake proofs can be minted.
* **Contract Audit:** Ensure the L1 smart contracts cannot be drained.

### 📅 Phase 10: Real L1/L2 Bridge & Mainnet Beta (v1.0.0)
*Connecting the Rollup to real Ethereum value.*
* **L1 Vaults:** Build the Ethereum Deposit Vaults that map 1:1 with L2 token balances. To guarantee absolute security, we will fork the battle-tested, open-source Canonical Bridge contracts from established ZK protocols (like Loopring) rather than writing security-critical deposit mechanics from scratch.
* **Mainnet Launch:** Deploy to Ethereum Mainnet with guarded deposit limits.

### 📅 Phase 11: Decentralized Provers (v2.0.0)
*Removing the final centralized component.*
* **Prover Market:** Allow anyone to run a Prover node and compete to generate proofs for the Sequencer in exchange for protocol rewards.

### 🔮 Phase 12: The Year 2 Crossroads (v3.0.0)
*After dominating the DEX market, the protocol decides its final form.*
* **Path A (The Ultimate App):** Remain a highly focused ZK-Rollup. We decentralize the Sequencer using a lightweight ordering consensus (like Espresso or Tendermint), keeping execution blazing fast and entirely focused on trading.
* **Path B (The Ecosystem Hub):** Pivot the massive TVL and userbase into a full Sovereign L1 Blockchain (Nowa Chain). The DEX becomes the flagship app, and developers worldwide are invited to build lending markets and NFT platforms using `$NOWA` as the native gas token.
