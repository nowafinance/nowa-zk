# TODO: Data Retention & Cleanup

## Current Problems

**Sequencer:**
- ❌ Storing all transactions forever (~1MB/batch)
- ❌ All batch data stored indefinitely

**Prover:**
- ❌ Saving proof binary (~1MB/batch) - not needed after verification
- ❌ Saving witness data (~10MB/batch) - not needed after proof generation
- ✅ Saving tx hash (32 bytes) and L2 tx hashes (4KB)

---

## Retention Strategy

### Sequencer (Smart Cleanup via ProverAPI)
- **Forever:** Batch metadata only (~200 bytes/batch: number, hash, state roots, timestamp)
- **Delete after L1 finalization:** Full transaction data once prover confirms L1 submission
- **Cleanup trigger:** Query ProverAPI to get latest finalized batch, delete all batches < finalized
- **Track cleanup:** Store `last_deleted_batch` to avoid redundant loops
- **Delete:** Mempool txs after batching, old batch full data after L1 proof

### Prover
- **Forever:** Proof TxHash, L2 TxHashes array, timestamp (~4.3KB/batch)
- **30 days:** Failure data for debugging
- **Delete immediately:** Proof binary, witness data, temp circuit data

**Storage Impact (100k batches):**
- Current (bad): ~1.1 TB (11MB/batch)
- Optimized (good): ~430 MB (4.3KB/batch)

**Why This Approach:**
- Prover has proof data on L1 (permanent)
- Sequencer doesn't need duplicate data after L1 finality
- Query prover prevents deleting unproven batches
- Tracks progress to avoid re-scanning old batches

---

## Configuration

Add to `.env`:
```bash
# Prover API endpoint (can run on separate machine)
ProverAPI=http://0.0.0.0:9091

# Optional cleanup settings
SEQUENCER_CLEANUP_INTERVAL_MINUTES=10
```

---

## Implementation Tasks

### Phase 1: Critical (Do Now)
- [x] Stop saving proof/witness binaries ✅
- [ ] Verify proof/witness not saved
- [ ] Add `ProverAPI` to `.env` and sequencer config
- [ ] Add sequencer `BatchMetadata` struct & table
- [ ] Store `last_deleted_batch` in sequencer DB

### Phase 2: Important (Next Week)
- [ ] Add prover API client in sequencer (`GetLatestProvenBatch()`)
- [ ] `CleanupOldBatches()` - delete batches < proven batch (`sequencer/internal/storage/store.go`)
- [ ] Track `last_deleted_batch` to avoid re-scanning
- [ ] `CleanupOldFailures()` - 30 day retention (`prover/internal/storage/store.go`)
- [ ] Cleanup failure files: `find ~/.tan-zk/prover/failures/ -type f -mtime +30 -delete`
- [ ] Add storage metrics (TotalBatches, TotalSizeBytes, OldestBatch, LastDeletedBatch)

### Phase 3: Nice to Have (Later)
- [ ] Schedule cleanup job (every 10 min, configurable)
- [ ] Disk space alerts (>80% usage)
- [ ] Configurable retention buffer (keep N batches after proven)

---

## Code Snippets

**Sequencer Metadata:**
```go
type BatchMetadata struct {
    Number, Timestamp int64
    Hash, OldStateRoot, NewStateRoot string
    TxCount int
}
```

**Prover API Client (in Sequencer):**
```go
type ProverClient struct {
    apiURL string
}

func (p *ProverClient) GetLatestProvenBatch() (uint64, error) {
    // GET ProverAPI/api/prover/latest-batch
    // Returns: {"batch_number": 1000, "tx_hash": "0x...", "proven": true}
}
```

**Smart Cleanup Logic:**
```go
func (s *Store) CleanupOldBatches(proverClient *ProverClient) error {
    // 1. Get last deleted batch from DB (avoid re-scanning)
    lastDeleted := s.GetLastDeletedBatch() // e.g., 900
    
    // 2. Query prover for latest proven batch on L1
    latestProven, err := proverClient.GetLatestProvenBatch() // e.g., 1000
    if err != nil { return err }
    
    // 3. Only process new batches since last cleanup
    for batchNum := lastDeleted + 1; batchNum < latestProven; batchNum++ {
        // Save metadata (if not already saved)
        s.SaveBatchMetadata(batchNum)
        
        // Delete full batch data
        s.DeleteFullBatch(batchNum)
        s.DeleteBatchTransactions(batchNum)
    }
    
    // 4. Update tracker
    s.SetLastDeletedBatch(latestProven - 1)
    return nil
}
```

---

## Testing

```bash
go test -run TestCleanupOldBatches  # sequencer
go test -run TestCleanupOldFailures # prover
```

Manual: Create 2000 batches → cleanup(1000) → verify counts

---

## Open Questions

- [ ] Archive to S3/IPFS instead of deleting?
- [ ] Actual batch finality time?
- [ ] Need 1000 full batches or less?
- [ ] Auto cleanup or manual flag?
