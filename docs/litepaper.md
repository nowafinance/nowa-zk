# Nowa-ZK Litepaper

## Abstract
Nowa-ZK is an App-Specific, ZK-verified Decentralized Exchange (DEX) built on a modular architecture: a Cosmos SDK-based chain as the Execution & Data Availability Layer, and Ethereum Sepolia as an L1 anchor that verifies trade-signature validity via Groth16 proofs. Nowa-ZK offers a high-performance execution environment secured by zero-knowledge cryptography.

---

## 1. Introduction
The demand for scalable blockchain solutions has never been higher, especially for orderbook-based trading platforms where users require lightning-fast execution and zero gas fees. General Purpose Rollups address some scalability issues but often require users to pay network gas fees and inherit bulky L1 DA costs.

Nowa-ZK introduces an **App-Specific, ZK-verified** approach. By executing trades on a fast, specialized Cosmos chain and submitting ZK proofs of trade-signature validity to Ethereum, Nowa-ZK anchors trade authenticity on-chain while keeping transaction costs essentially at zero for end users.

## 2. Architecture Overview

The Nowa-ZK architecture consists of four primary layers that interact seamlessly to provide a secure and fast transaction experience.

### 2.1 The Execution Layer (Cosmos Chain)
Users submit trades and EIP-712 signatures directly to the Nowa Cosmos-Ethereum blockchain. This chain acts as a highly specialized, fast-finality orderbook matching engine. Because trades happen here, users do not pay Ethereum gas fees.

### 2.2 The Data Availability (DA) Layer
In a traditional ZK Rollup, all trades are posted as `calldata` to Ethereum, which is prohibitively expensive for a high-frequency DEX. 
In Nowa-ZK, the **Cosmos chain itself acts as the Data Availability layer**. The history of all trades and balances is secured by the decentralized validator set of the Cosmos chain, drastically reducing L1 costs while maintaining transparency.

### 2.3 The Prover & Sequencer (Off-Chain Engine)
The Indexer and Prover (the core of this repository) act as the bridge between Cosmos and Ethereum.
*   **Indexer**: Continuously reads the Cosmos RPC, parsing new blocks and batching executed trades (e.g., 25 trades per batch).
*   **Prover**: A computational daemon that takes these batched trades and generates a succinct Groth16 Zero-Knowledge Proof (zk-SNARK). This proof mathematically guarantees that the trades were executed correctly and signatures are valid.

### 2.4 The Settlement Layer (Ethereum L1)
The Prover submits the final ZK Proof to the `TradeRegistry.sol` smart contract on Ethereum Sepolia.
*   **Signature Anchor**: `TradeVerifier.sol` mathematically verifies that each batch's trades carried valid signatures. This anchors trade authenticity to Ethereum; it does not track account balances or a state root on L1 — the Cosmos chain remains the source of truth for exchange state.
*   **Vaults**: A Smart Contract Vault on Ethereum for deposits/withdrawals is a standard pattern this architecture supports, distinct from the trade-verification path above.

---

## 3. Cryptographic Implementation (gnark)

Nowa-ZK leverages `gnark`, a highly optimized ZK-SNARK library written in Go.

### 3.1 ECDSA Signature Verification
To maintain full compatibility with Ethereum wallets (like MetaMask), the Cosmos chain uses standard ECDSA Secp256k1 signatures (EIP-712). The ZK circuit natively verifies these signatures inside the mathematical proof, ensuring that the Prover cannot fake trades.

### 3.2 Public Input Hashing (Scaling)
To circumvent Ethereum's EIP-170 smart contract size limit (24 KB), the circuit will utilize **Public Input Hashing**. Instead of exposing hundreds of trade hashes directly to the L1 Verifier, the circuit hashes all trades internally and outputs a single 32-byte hash. The Ethereum contract reconstructs this hash, allowing the system to verify thousands of trades per batch with minimal gas footprint.

## 4. Conclusion
By separating Execution (Cosmos), Data Availability (Cosmos), and Settlement (Ethereum), Nowa-ZK achieves the holy grail of decentralized trading: the speed and zero-fee experience of a centralized exchange, backed by the immutable, cryptographic security of Ethereum.
