# Nowa-ZK System Overview

## Architecture Diagram

```text
┌─────────────────────────────────────────────────────────────┐
│                         L1 (Ethereum)                        │
│  ┌────────────────────────────────────────────────────────┐ │
│  │          TradeRegistry Smart Contract                   │ │
│  │  • Receives trade metadata & ZK proofs                  │ │
│  │  • Verifies ZK proofs (via TradeVerifier)               │ │
│  │  • Manages canonical verified L2 state on L1            │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ Submit proofs & metadata
                              │
┌─────────────────────────────┴───────────────────────────────┐
│                         Prover                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  • Fetches batches from indexer                       │ │
│  │  • Generates ZK proofs (Groth16 via gnark)              │ │
│  │  • Submits proofs to L1                                 │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ Fetch batches
                              │
┌─────────────────────────────┴───────────────────────────────┐
│                       Indexer                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  • Indexes Cosmos Execution Layer (L2)                  │ │
│  │  • Groups trades into batches (e.g. 25 trades)          │ │
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
- **Batch Size**: Configurable (e.g., 25 trades per batch).
- **Storage**: Full batch data until L1 finalization.

### Prover
- **Purpose**: Generate cryptographic validity proofs for batches.
- **Proof System**: Groth16 on BN254 curve (`gnark`).
- **Submission**: Directly interacts with Ethereum L1 to anchor the Cosmos state.

### L1 Smart Contracts (Ethereum Sepolia)
- **Purpose**: Verify proofs and anchor the state.
- **Contracts**: `TradeRegistry` and `TradeVerifier`.
- **Verification**: On-chain Groth16 verification ensures the Prover cannot lie about the Cosmos state.

## Validium Data Flow Summary

1. **User** → Submits trade to the **Cosmos Execution Layer**.
2. **Indexer** → Groups 25 trades into a batch.
3. **Prover** → Generates a zk-SNARK proof.
4. **L1 Contract** → Verifies the proof and updates the anchored state on Ethereum.

## Key Features

### Scalability (Planned: Public Input Hashing)
- To bypass Ethereum's EIP-170 limit (24KB contract size), the circuit will utilize Public Input Hashing. The circuit compresses all trade data into a single hash, allowing batch sizes to scale from 25 to 1,000+ without increasing Ethereum gas costs.

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
