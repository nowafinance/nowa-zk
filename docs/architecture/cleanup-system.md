# Intelligent Cleanup System

## Overview

The cleanup system optimizes storage by automatically deleting old batch data from the indexer once batches have been proven and submitted to L1. This prevents unbounded storage growth while ensuring data is available when needed.

## Problem Statement

**Before Cleanup:**
- Indexer stores ALL batches forever
- Each batch: ~11 MB (128 full trades)
- 100,000 batches = ~1.1 TB storage
- Redundant data (prover has submitted to L1)

**With Cleanup:**
- Indexer deletes proven batches
- Only keeps recent unproven batches
- **98% storage reduction**

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    Indexer                              │
│                                                           │
│  ┌─────────────────────────────────────────────────────┐ │
│  │  Cleanup Job (runs every 5 minutes)                  │ │
│  │                                                       │ │
│  │  1. Get last_deleted_batch from DB                   │ │
│  │  2. Query ProverAPI/batches/latest                   │ │
│  │  3. Delete batches from last_deleted+1 to latest-1   │ │
│  │  4. Update last_deleted_batch tracker                │ │
│  └─────────────────────────────────────────────────────┘ │
│                          ▲                                │
│                          │ Queries every 5 min            │
└──────────────────────────┼───────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────┐
│                      Prover                               │
│                                                           │
│  API Endpoint: GET /batches/latest                        │
│                                                           │
│  Returns:                                                 │
│  {                                                        │
│    "batchNumber": 1000,                                   │
│    "txHash": "0x...",  ← L1 submission hash               │
│    "verifiedAt": 1234567890                               │
│  }                                                        │
└──────────────────────────────────────────────────────────┘
```

## Components

### 1. Prover API Endpoint

**File:** `prover/internal/api/server.go`

**Endpoint:** `GET /batches/latest`

Returns information about the latest batch proven and submitted to L1.

**Response:**
```json
{
  "batchNumber": 1000,
  "batchHash": "0x...",
  "txHash": "0x...",
  "verifiedAt": 1703001234
}
```

### 2. Indexer Prover Client

**File:** `indexer/internal/prover/client.go`

HTTP client that queries the prover API:

```go
type Client struct {
    apiURL     string
    httpClient *http.Client
}

func (c *Client) GetLatestProvenBatch() (uint64, error) {
    // GET ProverAPI/batches/latest
    // Returns latest proven batch number
}
```

### 3. Cleanup Tracker

**File:** `indexer/internal/indexer/store.go`

Tracks cleanup progress to avoid re-processing:

```go
// Get the batch number up to which data has been deleted
func (bs *BatchStore) GetLastDeletedBatch() (uint64, error)

// Update tracker after cleanup
func (bs *BatchStore) SetLastDeletedBatch(batchNum uint64) error

// Delete a specific batch's data
func (bs *BatchStore) DeleteBatch(batchNum uint64) error
```

**Database Key:** `cleanup:lastDeleted`

### 4. Cleanup Logic

**File:** `indexer/internal/indexer/cleanup.go`

Core cleanup algorithm:

```go
func (bs *BatchStore) CleanupOldBatches(proverClient ProverClient) error {
    // 1. Get last deleted batch (e.g., 900)
    lastDeleted := bs.GetLastDeletedBatch()
    
    // 2. Query prover for latest proven batch (e.g., 1000)
    latestProven := proverClient.GetLatestProvenBatch()
    if err != nil {
        // Prover unavailable - skip and retry later
        return nil
    }
    
    // 3. Delete batches 901 to 999
    for batchNum := lastDeleted + 1; batchNum < latestProven; batchNum++ {
        bs.DeleteBatch(batchNum)
    }
    
    // 4. Update tracker to 999
    bs.SetLastDeletedBatch(latestProven - 1)
    
    return nil
}
```

### 5. Scheduled Job

**File:** `indexer/internal/indexer/indexer.go`

Runs cleanup periodically:

```go
func (s *Service) runCleanupJob() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-s.ctx.Done():
            return
        case <-ticker.C:
            s.batches.CleanupOldBatches(proverClient)
        }
    }
}
```

## Configuration

**Environment Variable:** `PROVER_API`

**Example `.env`:**
```bash
# Prover API endpoint for cleanup coordination
PROVER_API=http://0.0.0.0:8081

# Use localhost if same machine
PROVER_API=http://localhost:8081

# Use remote IP if different server
PROVER_API=http://192.168.1.100:8081
```

**Default Value:** `http://localhost:8081`

## Cleanup Flow Example

### Initial State
```
Indexer DB: Batches 1-1000 (all stored)
Prover: Latest proven = 1000 (submitted to L1)
Tracker: last_deleted = 0
```

### Cleanup Run #1 (at T+5min)
```
1. Read tracker: last_deleted = 0
2. Query prover: latest_proven = 1000
3. Delete batches 1-999
4. Update tracker: last_deleted = 999

Result: Batches 1-999 deleted, batch 1000 kept (safety buffer)
```

### Cleanup Run #2 (at T+10min)
```
1. Read tracker: last_deleted = 999
2. Query prover: latest_proven = 1050
3. Delete batches 1000-1049
4. Update tracker: last_deleted = 1049

Result: Only processes 50 new batches (efficient!)
```

### Cleanup Run #3 (Prover Down)
```
1. Read tracker: last_deleted = 1049
2. Query prover: ERROR - connection refused
3. Log warning, skip cleanup
4. No changes to tracker

Result: Gracefully skips, will retry in 5 minutes
```

## Safety Mechanisms

### 1. Keep Latest Batch
Cleanup deletes up to `latestProven - 1`, keeping the latest proven batch.

**Reason:** Safety buffer against edge cases

### 2. Progress Tracking
The `last_deleted_batch` tracker ensures:
- No re-scanning of already deleted batches
- Efficient incremental cleanup
- Crash recovery (remembers progress)

### 3. Graceful Degradation
If prover is unavailable:
- ✅ Indexer continues operating normally
- ✅ Cleanup logs warning and skips
- ✅ Retries automatically in 5 minutes

### 4. No Data Loss
Cleanup only deletes batches **after** L1 submission confirmation from prover.

## Storage Impact

### Example: 100,000 Batches

**Without Cleanup:**
```
Indexer: 100,000 batches × 11 MB = 1.1 TB
Prover:    100,000 batches × 4 KB = 400 MB
Total:     1.1 TB
```

**With Cleanup (98,000 proven):**
```
Indexer: 2,000 recent batches × 11 MB = 22 GB
Prover:    100,000 metadata × 4 KB = 400 MB
Total:     22.4 GB
```

**Savings: ~98% reduction** (1.1 TB → 22.4 GB)

## Monitoring

### Cleanup Logs

**Successful Cleanup:**
```
🧹 Running scheduled cleanup...
🧹 Cleanup: Last deleted batch: 900
🧹 Cleanup: Latest proven batch: 1000
✅ Cleanup complete: deleted 99 batches (901 to 999)
```

**Prover Unavailable:**
```
🧹 Running scheduled cleanup...
🧹 Cleanup: Last deleted batch: 900
⚠️  Cleanup skipped: Unable to reach prover API (connection refused). Will retry later.
```

**Nothing to Cleanup:**
```
🧹 Running scheduled cleanup...
🧹 Cleanup: Last deleted batch: 999
🧹 Cleanup: Latest proven batch: 1000
🧹 No batches to cleanup (proven=1000, last_deleted=999)
```

### Status Endpoint

**Endpoint:** `GET /status`

Shows batch data range:

```json
{
  "status": "running",
  "batch_count": 1500,
  "oldest": 1001,
  "newest": 1500,
  "total_in_db": 500,
  "last_deleted": 1000
}
```

**Interpretation:**
- Database has 500 batches stored (1001-1500)
- Batches 1-1000 have been deleted (proven on L1)
- Latest batch is 1500

## Tuning Parameters

### Cleanup Interval

**Current:** 5 minutes

**Adjust in:** `indexer/internal/indexer/indexer.go`

```go
// Change from 5 minutes to desired value
ticker := time.NewTicker(5 * time.Minute)
```

**Recommendations:**
| Chain Activity | Interval | API Calls/Hour |
|---------------|----------|----------------|
| High (>100 batches/hour) | 3-5 min | 12-20 |
| Medium (10-100 batches/hour) | 5-10 min | 6-12 |
| Low (<10 batches/hour) | 10-30 min | 2-6 |
| Dev/Testing | 1-2 min | 30-60 |

### Safety Buffer

**Current:** Keeps latest proven batch (`latestProven - 1`)

**To keep more batches:**
```go
// Keep latest 10 proven batches
for batchNum := lastDeleted + 1; batchNum < latestProven - 10; batchNum++ {
    bs.DeleteBatch(batchNum)
}
```

## Future Enhancements

### Planned
- [ ] Configurable cleanup interval via env var (`CLEANUP_INTERVAL_MINUTES`)
- [ ] Metrics/Prometheus integration (batches deleted, cleanup duration)
- [ ] Archival option (compress old batches instead of deleting)

### Considered
- [ ] Cleanup on batch creation (instead of time-based)
- [ ] Prover-initiated push notifications (instead of polling)
- [ ] Tiered storage (hot/warm/cold based on age)

## Troubleshooting

### Cleanup Not Running

**Check logs for:**
```
🧹 Cleanup job scheduled (queries prover at http://0.0.0.0:8081 every 5 minutes)
🧹 Cleanup job started (interval: 5 minutes)
```

**If missing:**
- Verify `PROVER_API` is set in `.env`
- Restart indexer

### Prover Connection Issues

**Error:**
```
⚠️  Cleanup skipped: Unable to reach prover API (connection refused)
```

**Solutions:**
1. Check prover is running
2. Verify `PROVER_API` URL is correct
3. Test manually: `curl http://localhost:8081/batches/latest`
4. Check firewall/network (if remote prover)

### Batches Not Being Deleted

**Check:**
1. Is prover actually proving batches? (check prover logs)
2. Query prover: `curl http://localhost:8081/batches/latest`
3. Check indexer `/status` endpoint for `last_deleted`
4. Verify cleanup logs show deletion counts

## Related Documentation

- [Storage Architecture](./storage.md)
- [Indexer Details](./indexer.md)
- [Prover Details](./prover.md)
- [Data Flow](./data-flow.md)
