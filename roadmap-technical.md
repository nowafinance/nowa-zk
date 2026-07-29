# 🧠 Nowa-ZK: Internal Engineering Playbook

**Purpose:** This document is strictly for the internal engineering team (You and Gemini). It contains the highly technical, nuanced implementation details, edge cases, and cryptographic shortcuts required to actually build the system outlined in the public `ROADMAP.md`.

---

## 🟢 Phase 2: L1 Data Availability (DA) & ZK Scalability
*Goal: Upgrade the Rollup from a Validium (off-chain data) to a True ZK Rollup (on-chain data).*

### Technical Action Items
1. **Solidity Upgrade:** 
   - Open `TradeRegistry.sol`.
   - Modify the settlement function to require a new parameter: `Trade[] calldata _trades`.
   - This ensures the raw transaction history is permanently stored on Ethereum, allowing anyone to rebuild the Cosmos state if the sequencer crashes.
2. **Go Prover Upgrade:**
   - Update the Go Prover to construct this array and pass it alongside the ZK Proof during settlement.
3. **ZK Hash Compression (Fiat-Shamir):**
   - The Ethereum contract cannot process an array of 5,000 trades efficiently. 
   - The ZK Circuit must hash the `Trade[]` array down into a single 32-byte `PublicInput` hash. 
   - The Solidity contract will also hash the `calldata` array. If the Solidity Hash == the ZK Public Input Hash, the data is mathematically verified.

---

## 🟢 Phase 3: Universal Token State (The "Transfer" Circuit)
*Goal: Force Ethereum L1 to perfectly track the state of all Devnet wrapped tokens.*

### The "Genesis Root" Shortcut (CRITICAL DISCOVERY)
* **The Problem:** We discovered that the Nowa Devnet ERC-20 tokens (like `0x13bf1b07...`) were deployed using a custom constructor that emitted an `OwnershipTransferred` event, but **failed to emit a standard `Transfer` event**.
* **The Solution:** The Cosmos Indexer cannot automatically detect the initial supply by scanning Block 0. 
* **Action:** We will mathematically bypass Block 0. We will write a script to calculate a **Genesis State Root** (a Merkle Tree where the Faucet Address holds exactly 100 Billion of every token, and everyone else has 0). We hardcode this Genesis Root into Ethereum.

### Technical Action Items
1. **Ignore Minting:** Because the Devnet operates by having the Faucet *transfer* tokens to users, we **do not need a Mint Circuit yet**.
2. **Build `transfer_circuit.go`:**
   - The Cosmos Indexer will scan for `Transfer` events starting *after* the deployment block.
   - The ZK Circuit will verify the ECDSA signature of the transfer.
   - The Circuit will output a `NewStateRoot` which Ethereum will save.

---

## 🟢 Phase 4 & 5: The L1 Bridge & Escape Hatch
*Goal: Allow real money (Mainnet ETH) to enter the system trustlessly, and allow users to rescue their money if Cosmos crashes.*

### The Bridge Mechanics (L1 -> L2)
* **Insight:** Deposits DO NOT need ZK Proofs. 
* **Action:** We build `BridgeVault.sol`. When a user deposits 10 Real ETH on Ethereum, the Cosmos Indexer reads the event and triggers a Mint of 10 Wrapped ETH on Cosmos. 

### The Escape Hatch Mechanics (L2 -> L1)
* **Insight:** Withdrawals DO need ZK Proofs.
* **Action:** We build `burn_circuit.go` (or `withdrawal_circuit`). When a user burns Wrapped ETH on Cosmos, the Prover submits the proof to Ethereum, and `BridgeVault.sol` unlocks the Real ETH.

* **The Emergency Freeze:** We will code a 7-day timer into the Ethereum contract. If the Cosmos Sequencer crashes and no ZK proofs are submitted for 7 days, the contract enters **Emergency Mode**.
* **Trustless Rescue:** In Emergency Mode, users submit a standard Merkle Proof to Ethereum. Ethereum verifies their balance against the last frozen State Root and refunds their Real ETH automatically. No Cosmos node required.

### Multi-Chain Expansion
* **Action:** Once `BridgeVault.sol` is perfectly audited on Ethereum (Sepolia), we simply copy-paste the exact same Solidity contract to Binance Smart Chain, Arbitrum, and Polygon. The Cosmos Indexer will listen to all chains simultaneously.

---

## ⚠️ Core Engineering Principles
1. **Never Trust the Sequencer:** The Cosmos Sequencer is just a dumb database. The Ethereum Smart Contract must enforce the laws of physics using ZK Math.
2. **Calldata is King:** A ZK Rollup without Data Availability is just a centralized database. We must always post state diffs or full trades to L1.
