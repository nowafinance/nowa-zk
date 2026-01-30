# Sequencer Data Flow Architecture

## Overview

The Nowa-ZK sequencer processes blockchain transactions in a deterministic, crash-safe manner by fetching blocks in batches and creating fixed-size transaction batches.

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
        ├─ Extract all transactions from block
        ├─ Check for reorgs (compare block hash)
        └─ Store block hash for future checks
        
    Result: Array of 100 blocks with ALL transactions
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. TRANSACTION EXTRACTION (Deterministic Order)             │
└─────────────────────────────────────────────────────────────┘
                           ↓
    Process blocks IN ORDER (501, 502, 503...):
    
    Block #501: [tx1, tx2, tx3]      → Queue
    Block #502: [tx4, tx5]           → Queue
    Block #503: [tx6, tx7, tx8, tx9] → Queue
    ...
    
    Transaction Queue (in order):
    [tx1, tx2, tx3, tx4, tx5, tx6, tx7, tx8, tx9, ...]
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 4. BATCH CREATION (Exactly 128 transactions)                │
└─────────────────────────────────────────────────────────────┘
                           ↓
    Check for incomplete batch first:
    
    IF incomplete batch exists (e.g., 50/128):
       ├─ Fill it with next transactions (78 more)
       └─ Mark as complete at 128
    
    ELSE create new batch:
       ├─ Take next 128 transactions
       ├─ Compute batch hash
       ├─ Compute state root transition
       └─ Save to database
       
    IF < 128 transactions left:
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
Block #7501: 14 transactions
Block #7502: 16 transactions  
Block #7503: 12 transactions
Block #7504: 10 transactions
Block #7505: 500 transactions  ← Large block!
Block #7506: 15 transactions
...
Block #7600: 8 transactions
Total: 1,600+ transactions across 100 blocks
```

### Step 2: Process Transactions in Order
```
Queue (deterministic order):
[7501_tx1, 7501_tx2, ..., 7502_tx1, ..., 7505_tx1, 7505_tx2, ...]
                                    ↓ 500 txs from one block
```

### Step 3: Create Batches (128 each)
```
Batch #1:  128 txs from blocks 7501-7504
Batch #2:  128 txs from blocks 7504-7505 (partial)
Batch #3:  128 txs from block 7505
Batch #4:  128 txs from block 7505
Batch #5:  116 txs from block 7505 (remainder)
Batch #6:  12 txs from 7505 + 116 from 7506-7510
...
Batch #12: 95 txs (incomplete, waiting for more blocks)
```

### Step 4: Save Progress
```
Database state:
- last_processed_block: 7600
- batch_1 through batch_11: Complete (128 txs each)
- batch_12: Incomplete (95/128 txs)
- block_7501_hash through block_7600_hash: Saved
```

---

## Handling Large Blocks (128+ or 1000s of Transactions)

### ✅ **YES, the system handles this properly!**

**Example: Block with 1,000 transactions**

```go
// Processing loop in process_block_range.go
remainingTxs := block.txs // 1000 transactions

while len(remainingTxs) > 0 {
    if incomplete_batch exists {
        // Fill incomplete batch first
        fill_to_128()
    }
    
    if len(remainingTxs) >= 128 {
        // Create full batch
        batch = create_batch(remainingTxs[0:128])
        remainingTxs = remainingTxs[128:]  // 872 left
    } else {
        // Less than 128 left
        create_incomplete_batch(remainingTxs)
        remainingTxs = []  // Done
    }
}
```

**Result for 1000-tx block:**
- Batch #1: 128 txs (from this block)
- Batch #2: 128 txs (from this block)
- Batch #3: 128 txs (from this block)
- Batch #4: 128 txs (from this block)
- Batch #5: 128 txs (from this block)
- Batch #6: 128 txs (from this block)
- Batch #7: 128 txs (from this block)
- Batch #8: 104 txs (incomplete, from this block)

**Then next block's transactions continue filling Batch #8!**

---

## Key Guarantees

### 1. **Deterministic Ordering**
- Transactions always sorted by: `(block_number, transaction_index)`
- Same input blocks → Same batches
- Critical for proof reproducibility

### 2. **Crash Safety**
```
Scenario: Crash during processing blocks 501-600

On restart:
1. Load last_processed_block = 500
2. Load incomplete_batch (if any)
3. Resume from block 501
4. Same transaction order preserved
```

### 3. **No Transaction Loss**
- Every transaction from every block is included
- Batches never skip transactions
- Even if block fetch fails, entire 100-block range retries

### 4. **Exactly 128 Transactions Per Batch**
- No batch ever has more than 128 transactions
- Incomplete batches wait for more blocks
- One block can create multiple batches

---

## Database Schema

```
Keys stored in BadgerDB:

last_processed_block:   uint64              // Latest fully processed block
last_state_root:        string              // Current state root

block_<N>_hash:         string              // For reorg detection

batch_<N>:              {                   // Complete batch
  Number: uint64
  Transactions: [128]Tx
  Hash: string
  OldStateRoot: string
  NewStateRoot: string
  Timestamp: int64
}

batch_<N>_incomplete:   {                   // Incomplete batch
  Number: uint64
  Transactions: [1-127]Tx  // Less than 128
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
- Prevents transaction order corruption
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
| Batch size | 128 transactions | Fixed |
| Retry attempts | 3 per block | 2s, 4s, 6s backoff |
| Processing speed | ~5-6 blocks/sec | Depends on RPC |

**Example catch-up speed:**
- 10,000 blocks behind
- Processes 100 blocks every ~20 seconds
- Full catch-up: ~33 minutes

---

## Code References

- **Main loop:** [`sequencer/internal/sequencer/sequencer.go`](../../sequencer/internal/sequencer/sequencer.go)
- **Block processing:** [`sequencer/internal/sequencer/process_block_range.go`](../../sequencer/internal/sequencer/process_block_range.go)
- **Batch building:** [`sequencer/internal/sequencer/batch_builder.go`](../../sequencer/internal/sequencer/batch_builder.go)
- **Storage:** [`sequencer/internal/storage/store.go`](../../sequencer/internal/storage/store.go)
