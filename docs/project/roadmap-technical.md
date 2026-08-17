# 🧠 Nowa-ZK: Internal Engineering Playbook

**Purpose:** This document is strictly for the internal engineering team (You and Gemini). It contains the highly technical, nuanced implementation details, edge cases, and cryptographic shortcuts required to actually build the system outlined in the public `roadmap-marketing.md`.

> [!NOTE]
> For what's actually been published as a release vs. built-but-unreleased vs.
> genuinely incomplete right now, see [Release Status](./release-status.md).

---

## 🟢 Phase 3: Universal State Transition (Completed)
*Goal: Force Ethereum L1 to perfectly track the state of all users via a Sparse Merkle Tree.*

### Technical Action Items
1. **Merkle State:** We successfully implemented a Depth-20 Sparse Merkle Tree inside `StateTransitionCircuit`.
2. **State Updates:** Every trade now requires Merkle Inclusion Proofs for both the sender and receiver, mathematically guaranteeing they possess the required balances before altering the state root.

---

## 🟢 Phase 4: The High-Frequency Go Sequencer (built; WebSocket API still outstanding)
*Goal: Move execution away from a slow L2 blockchain to a high-frequency off-chain matching engine.*

### The "No L2 Blockchain" Epiphany
* **The Problem:** We originally planned to use a Cosmos or EVM blockchain as the execution layer. However, blockchains require consensus (block times), which severely limits high-frequency orderbook trading (e.g., 10,000 TPS).
* **The Solution:** We will operate exactly like dYdX v3 or Loopring. There is NO blockchain on Layer 2. The execution layer is simply a centralized, ultra-fast Go matching engine (The Sequencer).
* **Security Guarantee:** Even though the Sequencer is centralized, it **cannot steal funds or fake trades** because every state change it makes must be proven by our ZK-SNARK circuit and verified on Ethereum L1. If the Sequencer crashes, users trigger the Escape Hatch on L1.

### Technical Action Items
1. **The Orderbook:** Build an in-memory orderbook that matches makers and takers instantly.
2. **The Signature Pipeline (ECDSA):** We utilize standard Secp256k1 ECDSA signatures. Users sign their trades via MetaMask (EIP-712). 
3. **The Go Sequencer:** The Go Sequencer performs a lightning-fast ECDSA verification on incoming WebSocket trades for sub-millisecond soft finality. 
4. **The Merkle DB:** The Sequencer maintains a persistent database (LevelDB) representing the state of all balances. As trades match, it updates the DB and passes the state diffs (paths) and the signatures to the Prover.
5. **The ZK Batcher:** A background worker constantly polls the matched trades, groups them into batches of 25, and exposes a REST API for the Prover to fetch them.

---

## 🟡 Phase 5: The L1 Bridge & Escape Hatch — Bridge done, Escape Hatch NOT built
*Goal: Guarantee users can rescue their money if the Sequencer ever crashes, and allow real money to enter the system trustlessly.*

### The Bridge Mechanics (L1 -> L2) — ✅ Built
* **Action:** `NowaRollup.sol` is the canonical bridge. When a user deposits, the **Sequencer's own Deposit Watcher** (`sequencer/cmd/sequencer/deposit_watcher.go`, not the Indexer — that plan changed once the Sequencer replaced the Indexer as the execution layer) reads the `Deposit` event directly and triggers an `OpDeposit` transition.
* **Security note:** `deposit()`/`withdraw()` were not forked from Loopring/StarkEx as originally planned — they're a from-scratch, minimal implementation. `deposit()` is a straightforward `transferFrom` + event emit; `withdraw()` is currently just an owner-gated placeholder (see below), so this hasn't gone through the "battle-tested fork" hardening this section originally called for.

### The Escape Hatch Mechanics (L2 -> L1) — ❌ Not built
* **Status:** None of this exists yet. `NowaRollup.withdraw()` is `onlyOwner` and explicitly ignores its `_merkleProof` parameter (see the function's own doc comment: "Not an escape hatch"). There is no emergency-freeze timer, no Emergency Mode, and no path for a user to submit a Merkle proof and self-rescue funds if the Sequencer stops operating.
* **Why this matters:** every other phase's "trustless" framing depends on this existing. Until it's built, user funds' safety in a Sequencer-outage scenario rests entirely on the operator, not on-chain guarantees. This is the highest-priority gap in the whole project as of 2026-08-17.

---

## ⚠️ Core Engineering Principles
1. **Never Trust the Sequencer:** The Sequencer is just a fast matching engine. The Ethereum Smart Contract must enforce the laws of physics using ZK Math.
2. **Calldata is King:** A ZK Rollup without Data Availability is just a centralized database. We must always post state diffs (using EIP-4844 blobs) to L1.
