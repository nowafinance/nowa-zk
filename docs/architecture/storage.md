# Storage Architecture

## Overview

Nowa-ZK uses a tiered storage strategy to optimize for availability, efficiency, and cost.

## Storage Layers

### L2 Blockchain
- **Purpose**: Source of truth for recent trades
- **Technology**: Cosmos SDK LevelDB
- **Retention**: Full history (immutable)
- **Access**: RPC queries

### Indexer
- **Purpose**: Batch staging for proof generation
- **Technology**: BadgerDB (LSM tree)
- **Retention**: Recent unproven batches only
- **Cleanup**: Automatic deletion after L1 finalization

### Prover
- **Purpose**: Proof metadata and submission tracking
- **Technology**: BadgerDB
- **Retention**: Metadata only (~4KB per batch)
- **Data**: Batch number, L1 tx hash, L2 tx hashes, timestamp

### L1 Ethereum
- **Purpose**: Permanent state commitments
- **Technology**: Ethereum state trie
- **Retention**: Forever (state roots + batch commitments)
- **Data**: State roots, batch hashes, verification results

## Data Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│  L2 Transaction (Submitted)                                  │
│  • Stored on L2 blockchain forever                           │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  Indexer Batch Creation                                    │
│  • Full trade data stored (~11MB per batch)            │
│  • Kept until proven on L1                                   │
│  Retention: ~5-10 minutes (until proving)                    │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  Prover Processing                                           │
│  • Fetches batch via HTTP (no local storage)                │
│  • Generates proof in memory                                 │
│  • Submits to L1                                             │
│  • Saves only metadata (~4KB)                                │
│  Retention: Forever (minimal size)                           │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  L1 State Commitment                                         │
│  • State root stored on Ethereum                             │
│  • Batch commitment stored                                   │
│  Retention: Forever (blockchain immutable)                   │
└─────────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────────┐
│  Indexer Cleanup                                           │
│  • Deletes batch after L1 confirmation                       │
│  • Queries prover every 5 minutes                            │
│  • Keeps only recent unproven batches                        │
│  Result: ~98% storage reduction                              │
└─────────────────────────────────────────────────────────────┘
```

## Storage Breakdown

### Indexer Database Schema

**BadgerDB Keys:**

```
# Batches
batch_1              → {batch metadata + full tx data} (11MB)
batch_2              → {batch metadata + full tx data} (11MB)
...
batch_N              → {batch metadata + full tx data} (11MB)

# Block tracking (reorg detection)
block_hash_7500      → "0xabc..."
block_hash_7501      → "0xdef..."
...

# State tracking
last_processed_block → "7600"
cleanup:lastDeleted  → "900"
```

**Cleanup Impact:**
- Before: All batches stored (~1.1TB for 100k batches)
- After: Only recent unproven batches (~22GB for 2k batches)

### Prover Database Schema

**BadgerDB Keys:**

```
# Batch metadata
proof_1              → {batch_number, tx_hash, tx_hashes, timestamp} (4KB)
proof_2              → {batch_number, tx_hash, tx_hashes, timestamp} (4KB)
...
proof_N              → {batch_number, tx_hash, tx_hashes, timestamp} (4KB)

# State tracking
last_processed_batch → "1000"
last_state_root      → "0x123..."

# Failure tracking (optional)
failure_50           → {error, timestamp}
last_failure         → {batch_number, error}

# Halt state (safety)
halt_state           → {halted: true/false, reason}
```

**Total Size:** ~400MB for 100k batches

### L1 Blockchain Storage

**Smart Contract State:**

```solidity
// State variables
mapping(uint256 => bytes32) public batchHashes;
mapping(uint256 => bytes32) public stateRoots;
uint256 public latestBatch;
```

**On-Chain Data:** ~64 bytes per batch (2 state roots)

## Retention Policies

| Component | Data Type | Retention | Size per Batch |
|-----------|-----------|-----------|----------------|
| L2 Blockchain | Full trades | Forever | N/A (separate system) |
| Indexer | Full batch data | Until L1 proven | ~11 MB |
| Prover | Metadata only | Forever | ~4 KB |
| L1 Contract | State roots | Forever | ~64 bytes |

## Cleanup Strategy

See [Cleanup System](./cleanup-system.md) for detailed implementation.

### Key Points
- Indexer deletes batches after L1 proof submission
- Prover queries determine when it's safe to delete
- Cleanup runs every 5 minutes
- Tracks progress to avoid re-scanning

## Performance Characteristics

### Write Performance
| Component | Operation | Latency |
|-----------|-----------|---------|
| Indexer | Save batch | ~10ms |
| Prover | Save metadata | ~5ms |
| L1 Contract | Store state | ~12s (block time) |

### Read Performance
| Component | Operation | Latency |
|-----------|-----------|---------|
| Indexer | Get batch | ~5ms |
| Prover | Get metadata | ~3ms |
| L1 Contract | Read state | ~100ms (RPC) |

### Storage Growth Rate

**Example: 100 TPS sustained**
```
Batches per hour: 100 TPS ÷ 125 trades/batch × 3600s = ~2880 batches/hour

Without cleanup:
  2880 batches × 11 MB = ~31 GB/hour
  
With cleanup (keeping 2-3 hours):
  ~100 batches × 11 MB = ~1.1 GB steady state
```

## Archival Considerations

### Current Implementation
- No archival layer (L2 blockchain is source of truth)
- Indexer deletes proven batches
- Prover keeps lightweight metadata

### Future Options

**Option 1: S3/Cloud Storage**
- Archive deleted batches to S3
- Compress before upload (~90% reduction)
- Cost: ~$0.023/GB/month

**Option 2: IPFS/Arweave**
- Decentralized archival
- Content-addressed
- Pay once, store forever

**Option 3: Data Availability Layer**
- Use Celestia or EigenDA
- Designed for rollup data
- Cryptographic proofs of availability

## Disaster Recovery

### Indexer Recovery
1. Restart indexer → reads `last_processed_block`
2. Resumes from last checkpoint
3. Rebuilds incomplete batch from DB
4. Continues processing

### Prover Recovery
1. Restart prover → reads `last_processed_batch`
2. Queries indexer for next batch
3. Continues proving from checkpoint
4. No data loss (L1 is source of truth)

### Complete Data Loss
**Indexer:**
- Can rebuild from L2 blockchain (source of truth)
- Query L2 RPC for historical blocks
- Recreate batches deterministically

**Prover:**
- Cannot rebuild proofs (ephemeral)
- Metadata lost but not critical
- L1 still has state roots (verification unaffected)

## Database Configuration

### BadgerDB Tuning

**Indexer:**
```go
opts := badger.DefaultOptions(path)
opts.ValueLogFileSize = 256 << 20  // 256MB (large batches)
opts.NumLevelZeroTables = 5
opts.NumLevelZeroTablesStall = 10
```

**Prover:**
```go
opts := badger.DefaultOptions(path)
opts.ValueLogFileSize = 64 << 20   // 64MB (small metadata)
opts.NumCompactors = 2
```

## Monitoring

### Key Metrics
- Indexer DB size
- Prover DB size
- Batches deleted per cleanup run
- Cleanup duration
- Storage growth rate

### Alerts
- Indexer DB > 500GB (cleanup may be failing)
- Prover DB > 10GB (unexpected for metadata)
- Cleanup failing for > 30 minutes

## Related Documentation
- [Cleanup System](./cleanup-system.md)
- [Indexer Architecture](./indexer.md)
- [Prover Architecture](./prover.md)
