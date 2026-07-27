# Nowa ZK Orderbook - Architecture FAQ

This document outlines the high-level architecture decisions, cryptographic techniques, and rollup theories used in the Nowa ZK Orderbook project.

---

## 1. Is this a ZK Rollup?
Yes! This project is an **App-Specific ZK Rollup** (often referred to as an App-Chain or a Validium, depending on Data Availability).

Unlike "General Purpose" Rollups (like Arbitrum or zkSync) where anyone can deploy random smart contracts, an App-Specific Rollup is built for one specific purpose—in this case, a high-frequency trading DEX/Orderbook.

## 2. Why don't we have a native token or charge ETH for gas?
**General Purpose Rollups** (like zkSync) use ETH or their own token to charge gas for every single transaction to prevent spam. 

**App-Specific Rollups** (like dYdX or Loopring) often don't charge network gas to the user. Instead, the protocol subsidizes the L1 gas costs and makes revenue by taking a small trading fee (in USDC or a native token) on the trades. You have full control over the tokenomics. You can use ETH, your own token, or no token at all!

## 3. Where is the Bridge?
For a General Purpose Rollup, you need a massive, generalized L1 <-> L2 bridge. 
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
