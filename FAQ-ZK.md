# Nowa ZK Orderbook - Architecture FAQ

This document outlines the high-level architecture decisions, cryptographic techniques, and rollup theories used in the Nowa ZK Orderbook project.

---

## 1. Is this a ZK Rollup?
Yes! This project is an **App-Specific ZK Rollup** (often referred to as an App-Chain or a ZK Validium, depending on Data Availability). Famous examples of App-Specific ZK Rollups include **dYdX (v3)**, **Loopring**, and **Immutable X**.

Unlike "General Purpose" Rollups (like **Arbitrum** or **zkSync**) where anyone can deploy random smart contracts, an App-Specific Rollup is built for one specific purpose—in this case, a high-frequency trading DEX/Orderbook.

## 2. Tokenomics and Gas: If users pay gas on the Nowa Blockchain, is this still a ZK Rollup?
Yes, absolutely! To understand why, we need to clarify what a **"Native Token"** is and how gas works in different Rollup models.

In blockchain terminology, a **"Native Token"** is the fundamental token required to pay for transaction fees (gas) on a specific network. For example, ETH is the native token of Ethereum.

**General Purpose Rollups (like Arbitrum or zkSync):**
Because anyone can deploy any random smart contract on these networks, they *must* charge a gas fee for every transaction to prevent people from spamming the network. They usually use ETH or force users to use their own L2 native token for gas.

**App-Specific Rollups (like Nowa-ZK):**
As an App-Specific Rollup, the protocol has complete freedom over its tokenomics and user experience. 
Since trades execute on the **Nowa Blockchain** (a Cosmos-based chain), the network can choose to charge users a gas fee in the native Nowa token to prevent spam on the network. 

**Does charging gas on the Nowa Blockchain disqualify it from being a ZK Rollup?**
No! A system is defined as a ZK Rollup (or Validium) based on **how it secures its data**, not how it charges fees. 
Because the Prover takes the trades from the Nowa Blockchain, wraps them in a mathematical Zero-Knowledge Proof, and submits that proof to Ethereum (L1) for verification, the system is mathematically anchored to Ethereum. Ethereum does not dictate how fees are structured on the Cosmos side; it only verifies that the ZK Proof proves the state transitions correctly.

## 3. Where is the Bridge?
For a General Purpose Rollup, a massive, generalized L1 <-> L2 bridge is required. 
For an App-Specific ZK DEX, the "bridge" is simply a **Vault Contract** on Ethereum (L1). 
1. Users deposit USDC into the L1 Vault.
2. The Vault credits their off-chain L2 balance.
3. Users trade at lightning speed using this ZK engine.
4. Users request a withdrawal to pull their USDC back out of the Vault.

Currently, this repository contains the hardest part: **The Sequencer and Cryptographic Prover Engine**. The Vault/Bridge is a standard Solidity smart contract that can be added later.

## 4. What about Data Availability (DA)?
A "True ZK Rollup" posts every single trade and balance change directly to Ethereum L1 `calldata` so anyone can reconstruct the state. However, for a high-frequency DEX, this is far too expensive.

Because trades in this project happen on an underlying **Cosmos-Ethereum blockchain**, the Cosmos chain acts as the Data Availability layer. The validators of the Cosmos chain store the history. 
This makes the architecture a **ZK-Bridge** or **Validium** anchored to Ethereum. Ethereum mathematically verifies that the Cosmos chain is operating correctly via the `TradeVerifier`, creating a trustless connection.

## 5. What is "Public Input Hashing"?
When a ZK Proof is verified on Ethereum, the Smart Contract needs two things: the Proof itself, and the Public Inputs (the actual data being proven, like trade hashes).

If we process 25 trades, we have 25 trade hashes and 25 public keys. If we expose them all individually, we end up with 300+ Public Inputs. The Ethereum Smart Contract that verifies this (`TradeVerifier.sol`) becomes too massive and exceeds Ethereum's 24 KB contract size limit (EIP-170).

**The Solution:**
Instead of exposing 300 variables directly, the ZK Circuit uses a cryptographic hash function (like SHA256) to compress all the data into a single hash. 
- The circuit exposes **only 1 Public Input** (the final hash).
- The `TradeRegistry.sol` receives the trades, hashes them together, and passes that 1 hash to the Verifier.

This shrinks the Verifier contract to ~5 KB, allowing the system to process thousands of trades per batch without hitting Ethereum's size limits.

## 6. How do General Purpose Rollups handle Public Inputs?
Unlike App-Specific Rollups that verify specific actions (like trades), General Purpose Rollups (like zkSync or Scroll) operate as zkEVMs (Zero-Knowledge Ethereum Virtual Machines). Because they process arbitrary, random smart contract code, they never expose individual transaction details as public inputs.

Instead, they rely heavily on state roots and Public Input Hashing from the beginning. When a General Purpose Rollup submits a proof to Ethereum, the circuit typically exposes only three major public inputs:
1. **Old State Root** (The state of the network before the block).
2. **New State Root** (The state of the network after executing the block).
3. **Calldata Hash** (A single `keccak256` hash of all the raw transaction data in the block).

**How it works on L1:**
1. The Rollup posts all the raw, compressed transaction data to Ethereum L1 as `calldata` (to guarantee Data Availability).
2. The Ethereum L1 Smart Contract computes the `keccak256` hash of that data.
3. The Ethereum L1 Smart Contract passes that **1 single Hash** (plus the Old and New State Roots) into the ZK Verifier.
4. The ZK Proof proves: *"Starting with the Old State Root, executing these hashed transactions results in the New State Root."*

By implementing Public Input Hashing in Nowa-ZK, the protocol utilizes the exact same advanced cryptographic techniques that the largest general purpose rollups use to scale efficiently.

## 7. If the Cosmos chain halts or goes offline, what happens to user funds?
A true ZK Rollup guarantees **self-custody** by implementing an "Escape Hatch" (or Forced Withdrawal) on the L1 Smart Contract. 

If the Nowa Cosmos network experiences a catastrophic failure or the validators go offline, the L1 Ethereum contract is designed to detect that it hasn't received a ZK Proof within a specified timeout period. When this happens, the contract enters an "Emergency Mode." In this mode, users can bypass the Cosmos chain entirely and submit a Merkle proof of their L2 balance directly to the Ethereum smart contract to withdraw their USDC. This ensures that user funds can never be frozen or stolen by the rollup operators.

## 8. When is a trade actually final? On Cosmos or on Ethereum?
Nowa-ZK utilizes a dual-finality structure to provide both a Web2-like trading experience and Web3 security:
- **Soft Finality (The Execution Layer):** The moment a trade is executed on the Cosmos chain (which utilizes ~3 to 4 second block times), the transaction is considered finalized locally by the Tendermint consensus mechanism. The user's UI updates instantly, allowing them to execute consecutive trades without waiting for Ethereum.
- **Hard Finality (The Settlement Layer):** Every few minutes, the Prover scoops up a batch of those soft-finalized trades, generates a mathematical ZK Proof, and submits it to Ethereum L1. Once that Ethereum transaction confirms, the batch of trades is cryptographically immutable and irreversibly settled on Ethereum.

## 9. Does this require a Trusted Setup?
Yes, but **end-users do not need to do anything.** The Trusted Setup is a one-time cryptographic event coordinated by the protocol developers prior to launching on Mainnet.

Because Nowa-ZK utilizes the **Groth16** proving system on the **BN254** elliptic curve (via `gnark`), the cryptography requires the generation of a pair of "Master Keys": a Proving Key and a Verifying Key. 
To generate these keys securely, the protocol developers, auditors, and community members participate in a Multi-Party Computation (MPC) ceremony. Participants generate cryptographic randomness (often called "toxic waste"). As long as **at least one participant** in the entire ceremony permanently destroys their piece of the toxic waste, the Master Keys are perfectly secure, and it is mathematically impossible for anyone to forge fake ZK proofs. 

## 10. Why use a Cosmos blockchain for execution instead of a centralized server?
Many early-stage rollups use a single, centralized web-server to act as their "Sequencer." While this is fast, it creates a massive single point of failure and censorship risk.

By building the execution layer on the Cosmos SDK (utilizing `x/evm` and CometBFT consensus), the sequencing of trades is instantly decentralized. Instead of a single server, a global, distributed network of validators sequences the trades and secures the Data Availability. This architecture provides high-throughput execution while eliminating centralized bottlenecks.

## 11. How does Cosmos handle transactions and MEV (Maximal Extractable Value)?
By default, Cosmos (Tendermint) nodes order transactions in the mempool based on gas price (a priority mempool). This means that if a trader submits a transaction, a bot could theoretically pay a higher gas fee to jump ahead of them and front-run the trade—a common problem on standard Ethereum.

However, because Nowa-ZK is an **App-Chain**, the protocol developers have complete root-level control over the mempool logic and can implement advanced MEV solutions that are impossible on standard Ethereum. For example:
1. **Encrypted Mempools:** The network can hide transaction details from the validators until the block is executed, completely eliminating the ability for bots to front-run trades (similar to the Osmosis DEX).
2. **Protocol-Owned MEV:** The network can utilize software like **Skip Protocol** to capture MEV revenue at the protocol level. Instead of searcher bots extracting value from users, the protocol captures that value and redistributes it to the users or the DAO treasury.

## 12. Hardware & Cost Scaling: How much RAM and Gas is needed per batch?
For an App-Specific ZK Rollup like Nowa-ZK, the hardware requirements scale perfectly linearly with the number of trades in a batch. 

In Nowa-ZK, every single trade generates roughly `86,150` mathematical constraints (due to the heavy ECDSA signature verification). The underlying `gnark` Groth16 Prover requires approximately **2.5 GB of RAM per 1 Million constraints**.

**Nowa-ZK Scaling Table:**

| Trade Batch Size | Mathematical Constraints | Required RAM (Generating Keys) | Required RAM (Creating Proofs) | Ethereum L1 Gas Cost (Per Trade) |
| :--- | :--- | :--- | :--- | :--- |
| **25 Trades** *(Current)* | 2.15 Million | **~ 10.0 GB RAM** | **~ 5.5 GB RAM** | Medium-Low |
| **50 Trades** | 4.30 Million | **~ 20.0 GB RAM** | **~ 11.0 GB RAM** | Low |
| **100 Trades** | 8.61 Million | **~ 40.0 GB RAM** | **~ 22.0 GB RAM** | Very Low |
| **250 Trades** | 21.5 Million | **~ 100.0 GB RAM** | **~ 54.0 GB RAM** | Ultra Low |
| **500 Trades** *(Future-for mainnet- low gas fees on ethereum mainnet)* | 43.0 Million | **~ 200.0 GB RAM** | **~ 108.0 GB RAM** | Almost Zero |

> [!NOTE]
> **Scaling to 2000+ Trades using Recursive Proofs**
> Currently, the system uses a "monolithic" circuit structure. We are using a batch size of 25 trades for **Testnet** to keep server RAM requirements and cloud costs low. 
> 
> However, to scale to massive volumes without requiring Terabytes of RAM, the protocol will transition to **Recursive Proving** (generating ZK proofs of other ZK proofs). 
> *   **Future Testnet (256 Trades using Sequential Proving):** We will use **1 single Prover server**. It will sequentially process 8 batches of 32 trades one by one, then recursively aggregate those 8 small proofs into 1 final proof. This keeps both the RAM footprint and the server rental costs incredibly low for testing, trading a slightly slower finality time for maximum cost savings.
> *   **Future Mainnet (1024 - 2048 Trades using Parallel Proving):** We will deploy a cluster of **8 parallel Prover servers**. They will simultaneously process 8 batches of 128 to 256 trades at once, then recursively aggregate them. This drastically speeds up Ethereum finality time for production traders while capping the maximum RAM requirement of any single server.
> 
> This mathematical architecture allows Nowa-ZK to process thousands of trades in a single batch, distributing the heavy Ethereum L1 gas cost across 2000 users. This drives the per-trade fee to fractions of a penny, all while maintaining low RAM requirements across a distributed cluster of standard servers!
## 13. How fast is Sequential Proving? (Time vs RAM)
Zero-Knowledge cryptography heavily relies on algorithms like the Fast Fourier Transform (FFT). FFTs require your mathematical circuit to be exactly a **Power of 2** (e.g., $2^{20}$, $2^{21}$, $2^{22}$). If your trades generate `2,000,001` constraints, the Prover must artificially "pad" the circuit with dummy variables up to the next power of two (`4,194,304`), wasting 2 Million constraints of empty space!

By picking base batch sizes like **32, 64, or 128**, you fine-tune the math so it perfectly hugs the curve just below the next power of 2, ensuring 0% of your RAM or Prover time is wasted.

Assuming you rent a standard, single CPU server, here is an optimized table showing how long **1 Single Server** takes to generate an entire Recursive Tree sequentially (proving 8 batches one by one, then combining them):

| Base Batch Size | Total Trades (8 Batches) | RAM (Generating Keys) | RAM (Creating Proofs) | Time per Base Proof | Time for all 8 Batches | Time for Final Recursive Proof | **Total Time to Finality** |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **32 Trades** | 256 Trades | ~ 14 GB | **~ 8 GB** | ~ 20 Seconds | 160 Seconds | 30 Seconds | **~ 3.1 Minutes** |
| **64 Trades** | 512 Trades | ~ 30 GB | **~ 16 GB** | ~ 45 Seconds | 360 Seconds | 30 Seconds | **~ 6.5 Minutes** |
| **128 Trades** | 1024 Trades | ~ 60 GB | **~ 32 GB** | ~ 90 Seconds | 720 Seconds | 30 Seconds | **~ 12.5 Minutes** |
| **256 Trades** | 2048 Trades | ~ 120 GB | **~ 64 GB** | ~ 3.5 Minutes | 28 Minutes | 30 Seconds | **~ 28.5 Minutes** |

*(Note: The Final Recursive proof is very fast because verifying a SNARK inside a SNARK only takes a few million constraints, regardless of how many trades were inside the original SNARK).*

This shows that processing 1,024 trades sequentially on a single server only takes ~12.5 minutes to finalize on Ethereum, which is perfectly acceptable for a Rollup since the users already received instant "Soft Finality" on the Cosmos UI.


## 14. How do other major ZK Rollups compare in Hardware and Batch Sizes?
General Purpose Rollups (like **zkSync**, **Scroll**, and **Polygon zkEVM**) work very differently than App-Specific Rollups. Because they process arbitrary, random smart contract code (acting as a full zkEVM), their math is incredibly complex and requires massive server clusters or GPU farms to generate proofs. 

Instead of proving "trades", they prove "Ethereum Opcodes".

**General Purpose Rollups Comparison:**

| Rollup Network | Generating Keys (Trusted Setup) | Creating Proofs (RAM & Hardware) | Batch Size (Tx per Proof) | What Data do they submit to Ethereum L1? |
| :--- | :--- | :--- | :--- | :--- |
| **zkSync Era (Boojum)** | N/A (Transparent STARK setup) | 64 GB - 128 GB RAM + GPUs | Hundreds | State Diffs (Only final changed balances), State Roots, and ZK Proof. |
| **Polygon zkEVM** | N/A (Universal Setup, low RAM) | ~1 TB RAM (CPU) or massive GPU arrays | Hundreds | Full compressed Transaction Data, State Roots, and ZK Proof. |
| **Scroll** | N/A (Universal Setup, low RAM) | > 256 GB RAM + Heavy GPUs (Halo2) | Hundreds | Full Transaction Data (via EIP-4844 Blobs), State Roots, and ZK Proof. |
| **Starknet** | N/A (STARKs have no Trusted Setup) | > 128 GB RAM + Cloud Prover Farms | Thousands | State Diffs (using STARKs), State Roots, and the ZK Proof. |

**Key Takeaways:**
*   General Purpose Rollups require monumental hardware (often costing hundreds of thousands of dollars in cloud infrastructure) just to prove basic smart contract execution. 
*   By building Nowa-ZK as an **App-Specific Rollup**, we strip away all the zkEVM overhead. We only prove what matters (ECDSA signatures and balance transfers), allowing the Prover to run on standard servers while still achieving identical cryptographic security on Ethereum.

## 15. Why use Groth16 instead of a Universal Setup (like PLONK or Halo2)?
General Purpose Rollups (like zkSync or Scroll) often use Universal Setups (PLONK, Halo2) or STARKs. These systems are incredibly flexible because you don't have to generate new cryptographic keys if you change the smart contract code.

However, Nowa-ZK uses **Groth16**, which requires a strict, circuit-specific Trusted Setup. We chose Groth16 for three critical reasons:

1. **Cheapest L1 Gas Costs:** Groth16 is mathematically the absolute cheapest zero-knowledge proof to verify on Ethereum (costing only ~200,000 gas). STARKs and PLONK proofs are larger and much more expensive to verify. Since our primary goal is to drive trading fees to zero, minimizing L1 gas consumption is the ultimate priority.
2. **Circuit Stability:** Unlike a general-purpose network where anyone can deploy random smart contracts, Nowa-ZK is an App-Specific DEX. Our trading math (verify signature, adjust balance, update Merkle tree) is highly static. Since we won't be changing the core trading rules every week, the inconvenience of a one-time "Circuit-Specific Setup" before Mainnet is a non-issue.
3. **Battle-Tested Security:** Groth16 is the oldest, most battle-tested, and heavily audited proving system in the industry (securing networks like Zcash).
