# Nowa-ZK System Overview

## Architecture Diagram

```text
┌─────────────────────────────────────────────────────────────┐
│                         L1 (Ethereum)                        │
│  ┌────────────────────────────────────────────────────────┐ │
│  │          TradeRegistry Smart Contract                   │ │
│  │  • Receives plaintext trade data & ZK proofs, per chunk │ │
│  │  • Verifies ZK proofs (via TradeVerifier)               │ │
│  │  • Tracks isChunkVerified per (batch, chunk)            │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ Submit proofs, per 25-trade chunk
                              │
┌─────────────────────────────┴───────────────────────────────┐
│                         Prover                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  • Fetches 125-trade batches from indexer               │ │
│  │  • Splits each batch into 6 chunks of 25 trades         │ │
│  │  • Generates one Groth16 proof (gnark) per chunk        │ │
│  │  • Submits each chunk's proof to L1                     │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ Fetch batches
                              │
┌─────────────────────────────┴───────────────────────────────┐
│                       Indexer                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  • Indexes Cosmos Execution Layer (L2)                  │ │
│  │  • Groups trades into batches of 125                    │ │
│  │  • Provides API for prover                              │ │
│  │  • Cleanup: deletes proven batches (every 5 min)        │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘

                              ▲
                              │ Query blocks/txs
                              │
┌─────────────────────────────┴───────────────────────────────┐
│              Execution & DA Layer (Cosmos EVM)               │
│  • Specialized orderbook execution layer                    │
│  • Users submit trades directly to this chain               │
│  • Fast finality & acts as Data Availability (DA) layer     │
└─────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

### Execution & DA Layer (Cosmos)
- **Purpose**: Fast, zero-gas transaction execution and data availability.
- **Technology**: Cosmos SDK + EVM.
- **Block Time**: ~1 second.
- **Role**: This acts as the Validium data layer. Full state is secured by the Cosmos validators, preventing the need to post expensive calldata to Ethereum.

### Indexer
- **Purpose**: Batch L2 trades for proving.
- **Batch Size**: 125 trades per batch (`BatchSize` in indexer config).
- **Storage**: Full batch data until L1 finalization.

### Prover
- **Purpose**: Generate cryptographic validity proofs for each batch.
- **Proof System**: Groth16 on BN254 curve (`gnark`).
- **Chunking**: Each 125-trade batch is split into chunks of 25 trades (`circuits.TradeBatchSize`) — one Groth16 proof per chunk, so a batch takes up to 6 proofs/submissions to fully settle.
- **Submission**: Submits each chunk's proof directly to `TradeRegistry` on Ethereum L1.

### L1 Smart Contracts (Ethereum Sepolia)
- **Purpose**: Verify proofs and record which chunks have been settled.
- **Contracts**: `TradeRegistry` and `TradeVerifier`.
- **State kept on-chain**: `isChunkVerified[batchNumber][chunkIndex]` and `chunkBatchRoot[batchNumber][chunkIndex]` — verification is tracked per chunk, not per whole batch.
- **Verification**: On-chain Groth16 verification ensures the Prover cannot lie about which signed trades existed.

## Validium Data Flow Summary

1. **User** → Submits trade to the **Cosmos Execution Layer**.
2. **Indexer** → Groups 125 trades into a batch.
3. **Prover** → Splits the batch into chunks of 25 trades and generates one zk-SNARK proof per chunk.
4. **L1 Contract** → Verifies each chunk's proof and marks that chunk settled.

## Key Features

### Fitting Under Ethereum's Contract Size Limit
- Exposing every trade's hash and public key as individual public inputs would push `TradeVerifier.sol` past Ethereum's 24 KB contract size limit (EIP-170). Instead, the circuit folds all 25 trades' `(messageHash, pubKeyX, pubKeyY)` into a single MiMC hash (`BatchRoot`), and only that one hash is exposed as a public input. `TradeRegistry.sol` recomputes the same MiMC hash on-chain from the plaintext trade data submitted alongside the proof, and checks it matches before calling the verifier.

### Security
- L1 inherits Ethereum's cryptoeconomic security.
- ZK proofs ensure trades were valid and EIP-712 signatures matched.

## Technology Stack

| Component | Technologies |
|-----------|-------------|
| Execution Layer | Cosmos SDK, EVM module |
| Indexer | Go, BadgerDB, Fiber API |
| Prover | Go, gnark (Groth16), BadgerDB |
| Smart Contracts | Solidity, Foundry |
| ZK Circuit | gnark constraint system |
