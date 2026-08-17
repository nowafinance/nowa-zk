# Nowa ZK Orderbook - Architecture FAQ

This document outlines the high-level architecture decisions, cryptographic techniques, and rollup theories used in the Nowa ZK Orderbook project.

---

## 1. Is this a ZK Rollup or a Validium?
This project is currently operating as a **ZK Validium**, but is actively transitioning into a **True ZK-Rollup**. 

To understand the difference, look at how the industry classifies the largest DEXs in the world based on **Data Availability (DA)**:

### ⚡ Validium DEXs (Data is Off-Chain)
These DEXs use ZK Proofs for security, but keep their massive arrays of trade data off-chain (often managed by Data Availability Committees) to save money on Ethereum gas fees and achieve extreme scalability. While often broadly grouped under the "ZK-Rollup" umbrella in general industry terminology, technical researchers classify them as Validiums.
*   **dYdX (v3):** One of the largest perpetual DEXs in the world. Built on StarkEx, it operated as a pure Validium.
*   **Rhino.fi (formerly DeversiFi):** A major spot-trading DEX built on StarkEx using the Validium model.
*   **ApeX Pro:** A derivatives DEX utilizing the StarkEx Validium engine.
*   **Immutable X:** An NFT exchange that uses the exact same Validium matching engine logic.

### ✅ True ZK-Rollup DEXs (Data is On-Chain)
These DEXs post their raw trade data directly to Ethereum (`calldata` or `blobs`). Even if their servers crash and burn, anyone in the world can download the Ethereum history and mathematically reconstruct the database to rescue their funds.
*   **Loopring:** One of the oldest and most famous order-book DEXs on Ethereum. They are a **Pure ZK-Rollup** and put every single trade into `calldata`.
*   **SyncSwap / Mute.io:** DEXs built on **zkSync Era**. Because zkSync is a pure ZK-Rollup, the DEXs inherit that on-chain data security.
*   **JediSwap / Ekubo:** Built on **Starknet**, which is a pure ZK-Rollup.
*   **QuickSwap:** Built on **Polygon zkEVM** (pure ZK-Rollup).

**Where does Nowa-ZK fit?**
Currently, Nowa-ZK is in the **dYdX (v3)** category (Validium). Once Data Availability (Phase 2) is implemented to post trade data to L1 `calldata`, Nowa-ZK will instantly graduate into the **Loopring** category (True ZK-Rollup).

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

In Nowa-ZK, every single trade generates a highly optimized number of mathematical constraints (due to the extremely lightweight EdDSA signature verification). The underlying `gnark` Groth16 Prover requires approximately **2.5 GB of RAM per 1 Million constraints**.

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

## 16. App-Specific vs General Purpose (Cost & Architecture)
Building a General Purpose zkEVM (like zkSync or Scroll) is one of the most notoriously expensive and difficult engineering feats in the world. By choosing to build an **App-Specific ZK Validium/Rollup**, the protocol strips away 99% of the unnecessary bloat required to prove random smart contracts.

| Metric | App-Specific Validium (Nowa-ZK) | General Purpose zkEVM (e.g. zkSync / Scroll) |
| :--- | :--- | :--- |
| **Primary Goal** | Hyper-optimized for exactly one thing (Trading). | Turing Complete (Anyone can deploy any smart contract). |
| **Engineering Team** | 1-3 standard Go/Solidity Developers. | 30-50+ Cryptographers, Compiler Engineers, and Rust Developers. |
| **Time to Market** | A few months. | 2 to 4 years of intense R&D. |
| **Prover Hardware Requirements** | Standard CPU Servers (8GB to 64GB RAM). | Massive GPU Farms or Supercomputers (1TB+ RAM). |
| **Estimated Monthly Server Costs** | **~$50 - $500 / month** | **$50,000 - $250,000+ / month** |
| **Circuit Complexity** | Very Low (~2M constraints per batch). | Incredibly High (Hundreds of Millions of constraints to prove Ethereum Opcodes). |
| **Ethereum L1 Gas Cost** | **~250,000 to 300,000 Gas per batch** (Pure Groth16 Verification, 0 calldata costs). | **~500,000 to 1,500,000+ Gas per batch** (Massive Data Availability blob/calldata costs). |
| **Security Risk** | **Low:** The math is static and hardcoded. Users can only trade. | **High:** Users can deploy malicious smart contracts or exploit the EVM engine. |

## 17. Are we submitting the user's trade signatures to Ethereum L1?
No! And that is exactly how a ZK-Rollup is supposed to work.

If this were an **Optimistic Rollup**, we would have to submit the raw signatures (`v, r, s` values) to the Ethereum `calldata` so that validators could verify the trades later if there was a dispute. Signatures take up a huge amount of bytes, which makes Optimistic Rollups expensive.

Because Nowa-ZK is a **ZK-Rollup**, we only submit two things to L1:
1. The **Trade Data** (`messageHash`, `pubKeyX`, `pubKeyY`) so that the state can be reconstructed (Data Availability).
2. The **ZK Proof**.

The ZK Prover (the Go codebase) takes the user's signature as a **private input** inside the Groth16 circuit. It mathematically verifies the EdDSA signature inside the circuit, but it *never* reveals the signature to Ethereum. The Ethereum smart contract just verifies the tiny 8-element ZK Proof. Because the proof is mathematically valid, Ethereum knows with 100% certainty that valid signatures existed for every single trade in that batch.

By *not* sending the signatures to L1, the protocol saves massive amounts of gas and keeps the L1 footprint incredibly small.

## 18. Why use EdDSA instead of Ethereum's default ECDSA?
Ethereum natively uses **ECDSA** (specifically on the secp256k1 curve) for all wallet signatures. While it works perfectly on Layer 1, verifying an ECDSA signature *inside* a Zero-Knowledge circuit is incredibly expensive.

Verifying a single ECDSA signature inside a ZK circuit requires roughly **1,500,000+ constraints** because the circuit has to emulate the entire secp256k1 math in a different mathematical field. If a batch contains 100 trades, that would be 150 Million constraints just for the signatures, requiring Terabytes of RAM!

**The EdDSA Solution:**
Instead of using ECDSA, modern ZK Rollups use **EdDSA (Edwards-curve Digital Signature Algorithm)** paired with a ZK-friendly hash function like **MiMC** or **Poseidon**. Verifying an EdDSA signature on the BN254 curve inside a `gnark` Groth16 circuit only takes a few thousand constraints—making it over **100x cheaper and faster** to prove than ECDSA.

**Who else uses EdDSA?**
Every major ZK Rollup that requires high-performance signature verification uses EdDSA (or similar ZK-friendly variants):
*   **dYdX & StarkEx DEXs:** Uses STARK-friendly signatures (similar to EdDSA) derived from the user's Ethereum wallet.
*   **Loopring:** Generates an EdDSA keypair derived from the user's MetaMask signature for all L2 trades.
*   **zkSync:** While zkSync Era abstracts some of this away, early versions heavily relied on specific ZK-friendly signatures for cheap transaction verification.

By switching to EdDSA for Nowa-ZK, the protocol ensures lightning-fast prover times, incredibly low RAM requirements, and production-level scalability.
