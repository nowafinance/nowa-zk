# Nowa-ZK System Overview

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         L1 (Ethereum)                        │
│  ┌────────────────────────────────────────────────────────┐ │
│  │          BatchRegistry Smart Contract                   │ │
│  │  • Stores batch commitments & state roots               │ │
│  │  • Verifies ZK proofs                                   │ │
│  │  • Manages L2 state on L1                               │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ Submit proofs
                              │
┌─────────────────────────────┴───────────────────────────────┐
│                         Prover                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  • Fetches batches from indexer                       │ │
│  │  • Generates ZK proofs (Groth16)                        │ │
│  │  • Submits proofs to L1                                 │ │
│  │  • Stores metadata (tx hashes only, ~4KB per batch)     │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ Fetch batches
                              │ Cleanup queries
┌─────────────────────────────┴───────────────────────────────┐
│                       Indexer                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  • Indexes L2 blockchain                                │ │
│  │  • Groups transactions into batches (128 txs)           │ │
│  │  • Provides API for prover                              │ │
│  │  • Cleanup: deletes proven batches (every 5 min)        │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              ▲
                              │ Query blocks/txs
                              │
┌─────────────────────────────┴───────────────────────────────┐
│                    L2 Blockchain (Cosmos EVM)                │
│  • EVM-compatible execution layer                           │
│  • Users submit transactions to L2                          │
│  • Fast finality (~1s block time)                           │
└─────────────────────────────────────────────────────────────┘
```

## Component Responsibilities

### L2 Blockchain
- **Purpose**: Fast, low-cost transaction execution
- **Technology**: Cosmos SDK + EVM module
- **Block Time**: ~1 second
- **Finality**: Instant soft finality, proven on L1

### Indexer
- **Purpose**: Batch L2 transactions for proving
- **Batch Size**: 128 transactions per batch
- **Storage**: Full batch data until L1 finalization
- **Cleanup**: Automatic deletion after prover confirms L1 submission

### Prover
- **Purpose**: Generate validity proofs for batches
- **Proof System**: Groth16 on BN254 curve
- **Storage**: Metadata only (no proof/witness, no full tx data)
- **Submission**: Direct to L1 BatchRegistry contract

### L1 Smart Contracts
- **Purpose**: Verify proofs and manage L2 state
- **Contract**: BatchRegistry
- **Verification**: On-chain Groth16 verifier
- **State**: Tracks state roots and batch commitments

## Data Flow Summary

1. **User** → Submits transaction to **L2 Blockchain**
2. **Indexer** → Indexes L2 blocks, groups into batches
3. **Prover** → Fetches batch, generates proof, submits to L1
4. **L1 Contract** → Verifies proof, updates state root
5. **Indexer** → Queries prover, deletes old batch data (cleanup)

See [Data Flow](./data-flow.md) for detailed transaction lifecycle.

## Key Features

### Scalability
- Off-chain execution (L2 handles TPS)
- Batched proving (128 txs proved together)
- Constant L1 verification cost per batch

### Data Efficiency
- Indexer: Stores batches temporarily until proven
- Prover: Only stores metadata (~4KB per batch)
- Cleanup: Automatic deletion based on L1 finality

### Security
- L1 inherits Ethereum security
- ZK proofs ensure validity
- No trust in indexer or prover (verifiable)

## Storage Optimization

### Before Cleanup
- Indexer stores all batches (~11MB each)
- Storage grows indefinitely

### With Cleanup System
- Indexer deletes batches after L1 finalization
- Query prover every 5 minutes for latest proven batch
- Only keeps recent unproven batches
- **Result**: ~98% storage reduction

See [Cleanup System](./cleanup-system.md) for implementation details.

## Network Topology

### Centralized (Current)
```
L2 Blockchain
    ↓
Indexer (single instance)
    ↓
Prover (single instance)
    ↓
L1 Ethereum
```

### Distributed (Future)
```
L2 Blockchain
    ↓
Multiple Indexers (redundancy)
    ↓
Prover Pool (load balancing)
    ↓
L1 Ethereum
```

## Technology Stack

| Component | Technologies |
|-----------|-------------|
| L2 Blockchain | Cosmos SDK, EVM module |
| Indexer | Go, BadgerDB, Fiber API |
| Prover | Go, gnark (Groth16), BadgerDB |
| Smart Contracts | Solidity, Foundry |
| ZK Circuit | gnark constraint system |

## Performance Characteristics

| Metric | Value |
|--------|-------|
| L2 Block Time | ~1 second |
| Batch Size | 128 transactions |
| Proof Generation | ~2-5 minutes per batch |
| L1 Submission | ~1 transaction per batch |
| Storage per Batch | Indexer: 0 (deleted), Prover: 4KB |
| Cleanup Frequency | Every 5 minutes |

## Next Steps

- Read [Indexer Architecture](./indexer.md)
- Read [Prover Architecture](./prover.md)
- Read [Cleanup System](./cleanup-system.md)
- Review [Data Flow](./data-flow.md)
