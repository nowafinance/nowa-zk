# Indexer Data Flow Architecture

## Overview

The Nowa-ZK indexer processes blockchain transactions in a deterministic, crash-safe manner by fetching blocks in batches and creating fixed-size trade batches.

Note: this document covers the **indexer's** batch (125 trades). The **prover** further splits each indexer batch into 6 chunks of 25 trades — one Groth16 proof per chunk — before submitting to L1. That chunking is a prover-side concern and isn't shown here; see `docs/architecture/overview.md`.

---

## Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│ 1. POLLING (Every 2 seconds)                                │
└─────────────────────────────────────────────────────────────┘
                           ↓
    Get current block number from RPC (e.g., block #9500)
    Last processed: #500
    Behind by: 9000 blocks
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. FETCH BLOCKS (100 at a time)                             │
└─────────────────────────────────────────────────────────────┘
                           ↓
    Process blocks #501-600 together:
    
    FOR each block in [501...600]:
        ┌─ Fetch block via RPC (with 3 retries: 2s, 4s, 6s)
        ├─ Extract and decode all trades from block transactions
        ├─ Check for reorgs (compare block hash)
        └─ Store block hash for future checks
        
    Result: Array of 100 blocks with ALL trades
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. TRADE EXTRACTION (Deterministic Order)                   │
└─────────────────────────────────────────────────────────────┘
                           ↓
    Process blocks IN ORDER (501, 502, 503...):
    
    Block #501: [trade1, trade2, trade3]      → Queue
    Block #502: [trade4, trade5]              → Queue
    Block #503: [trade6, trade7, trade8]      → Queue
    ...
    
    Trade Queue (in order):
    [trade1, trade2, trade3, trade4, trade5, trade6, trade7, trade8, ...]
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. BATCH CREATION (Exactly 125 trades)                       │
└─────────────────────────────────────────────────────────────┘
                           ↓
    Check for incomplete batch first:
    
    IF incomplete batch exists (e.g., 40/125):
       ├─ Fill it with next trades (85 more)
       └─ Mark as complete at 125
    
    ELSE create new batch:
       ├─ Take next 125 trades
       ├─ Compute batch hash
       └─ Save to database
       
    IF < 125 trades left:
       └─ Create incomplete batch, wait for more blocks
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 5. STATE PERSISTENCE                                         │
└─────────────────────────────────────────────────────────────┘
                           ↓
    After successfully processing blocks 501-600:
    
    ✅ Save last_processed_block = 600
    ✅ Save incomplete batch (if any)
    ✅ Save block hashes for reorg detection
    ✅ Notify API/WebSocket subscribers
                           ↓
    Continue to next range: 601-700 →
```

---

## Example Scenario

**Processing blocks 7501-7600:**

### Step 1: Fetch 100 Blocks
```
Block #7501: 50 trades
Block #7502: 40 trades
Block #7503: 300 trades  ← Large block!
Block #7504: 10 trades
...
Block #7600: 5 trades
Total: 1,000+ trades across 100 blocks
```

### Step 2: Process Trades in Order
```
Queue (deterministic order):
[7501_trade1, ..., 7502_trade1, ..., 7503_trade1, 7503_trade2, ...]
                                     ↓ 300 trades from one block
```

### Step 3: Create Batches (125 each)
```
Batch #1:  125 trades = 50 (block 7501) + 40 (block 7502) + 35 (partial, block 7503)
Batch #2:  125 trades from block 7503 (partial)
Batch #3:  125 trades from block 7503 (remainder: 265 - 125 - 125 = 15 left over)
Batch #4:  25 trades so far (15 from block 7503 + 10 from block 7504), incomplete, waiting for more blocks
...
```

### Step 4: Save Progress
```
Database state:
- last_processed_block: 7600
- batch_1 through batch_3: Complete (125 trades each)
- batch_4: Incomplete (25/125 trades, growing as later blocks are processed)
- block_7501_hash through block_7600_hash: Saved
```

---

## Handling Large Blocks (125+ or 1000s of Trades)

### ✅ **YES, the system handles this properly!**

**Example: Block with 1,030 trades**

```go
// Processing loop in process_block_range.go
remainingTrades := block.trades // 1030 trades

while len(remainingTrades) > 0 {
    if incomplete_batch exists {
        // Fill incomplete batch first
        fill_to_125()
    }
    
    if len(remainingTrades) >= 125 {
        // Create full batch
        batch = create_batch(remainingTrades[0:125])
        remainingTrades = remainingTrades[125:]
    } else {
        // Less than 125 left
        create_incomplete_batch(remainingTrades)
        remainingTrades = []  // Done
    }
}
```

**Result for 1030-trade block:**
- Batch #1: 125 trades (from this block)
- Batch #2: 125 trades (from this block)
- Batch #3: 125 trades (from this block)
...
- Batch #8: 125 trades (from this block) — 1000 trades used, 30 remain
- Batch #9: 30 trades (from this block), incomplete

**Then next block's trades continue filling Batch #9!**

---

## Key Guarantees

### 1. **Deterministic Ordering**
- Trades always sorted by: `(block_number, transaction_index, log_index)`
- Same input blocks → Same batches
- Critical for proof reproducibility

### 2. **Crash Safety**
```
Scenario: Crash during processing blocks 501-600

On restart:
1. Load last_processed_block = 500
2. Load incomplete_batch (if any)
3. Resume from block 501
4. Same trade order preserved
```

### 3. **No Trade Loss**
- Every valid trade from every block is included
- Batches never skip trades
- Even if block fetch fails, entire 100-block range retries

### 4. **Exactly 125 Trades Per Batch**
- No batch ever has more than 125 trades
- Incomplete batches wait for more blocks
- One block can create multiple batches

---

## Database Schema

```
Keys stored in BadgerDB:

last_processed_block:   uint64              // Latest fully processed block

block_<N>_hash:         string              // For reorg detection

batch_<N>:              {                   // Complete batch
  Number: uint64
  Trades: [125]ParsedTrade
  Hash: string
  OldStateRoot: string  // Hardcoded 0x0
  NewStateRoot: string  // Hardcoded 0x0
  Timestamp: int64
}

batch_<N>_incomplete:   {                   // Incomplete batch
  Number: uint64
  Trades: [1-124]ParsedTrade  // Less than 125
  ...
}
```

---

## Error Handling

### Block Fetch Failures
```
Attempt 1: Fail → Wait 2s
Attempt 2: Fail → Wait 4s  
Attempt 3: Fail → Wait 6s
Attempt 4: Fail → ❌ Entire 100-block range fails

On next poll (2s later):
→ Retry entire 100-block range from start
```

**Why fail entire range?**
- Prevents trade order corruption
- If block 505 fails but 506 succeeds, skipping 505 would break order
- Better to retry all 100 blocks than have wrong sequence

### Reorg Detection
```
Stored:  block_7500_hash = 0xabcd...
Fetched: block_7500_hash = 0xef12... ← Different!

Actions:
1. Find fork point (common ancestor)
2. Delete all batches after fork point
3. Rollback state to fork point
4. Re-process from fork point
```

---

## Performance Characteristics

| Metric | Value | Notes |
|--------|-------|-------|
| Blocks per fetch | 100 | Configurable |
| Poll interval | 2 seconds | Constant |
| Batch size | 125 trades | Fixed |
| Retry attempts | 3 per block | 2s, 4s, 6s backoff |
| Processing speed | ~5-6 blocks/sec | Depends on RPC |

**Example catch-up speed:**
- 10,000 blocks behind
- Processes 100 blocks every ~20 seconds
- Full catch-up: ~33 minutes

---

## Code References

- **Main loop:** [`indexer/internal/indexer/indexer.go`](../../indexer/internal/indexer/indexer.go)
- **Batch building:** [`indexer/internal/indexer/batch.go`](../../indexer/internal/indexer/batch.go)
- **Storage:** [`indexer/internal/indexer/store.go`](../../indexer/internal/indexer/store.go)
