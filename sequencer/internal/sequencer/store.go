package sequencer

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/dgraph-io/badger/v4"
	"github.com/tannetwork/zk-sequencer/sequencer/internal/sequencer/types"
)

// BatchStore stores batches locally
type BatchStore struct {
	mu      sync.RWMutex
	path    string
	batches map[uint64]*types.Batch
	count   uint64
	db      *badger.DB
}

// NewBatchStore creates a new batch store
func NewBatchStore(path string) (*BatchStore, error) {
	// Create directory if it doesn't exist
	os.MkdirAll(path, 0755)

	// Open BadgerDB (use subdirectory for DB files)
	dbPath := filepath.Join(path, "db")
	db, err := badger.Open(badger.DefaultOptions(dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open badger DB: %w", err)
	}

	store := &BatchStore{
		path:    path,
		batches: make(map[uint64]*types.Batch),
		db:      db,
	}

	// Load existing batches (from BadgerDB, not JSON files)
	if err := store.loadBatches(); err != nil {
		db.Close() // Clean up on error
		return nil, fmt.Errorf("failed to load batches: %w", err)
	}

	return store, nil
}

// SaveBatch saves a batch to disk
func (bs *BatchStore) SaveBatch(batch *types.Batch) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Store in memory
	bs.batches[batch.Number] = batch
	if batch.Number > bs.count {
		bs.count = batch.Number
	}
	// Marshal batch to JSON
	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	// Create key with "batch:" prefix to match loadBatches
	key := []byte(fmt.Sprintf("batch:%d", batch.Number))

	// Save to BadgerDB
	err = bs.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, data)
	})

	if err != nil {
		return fmt.Errorf("failed to set batch: %w", err)
	}

	return nil
}

// GetBatch retrieves a batch by number
func (bs *BatchStore) GetBatch(number uint64) (*types.Batch, error) {
	bs.mu.RLock()
	// Check in-memory cache first
	if batch, ok := bs.batches[number]; ok {
		bs.mu.RUnlock()
		return batch, nil
	}
	bs.mu.RUnlock()

	// If not in cache, read from BadgerDB
	var batch *types.Batch
	err := bs.db.View(func(txn *badger.Txn) error {
		key := []byte(fmt.Sprintf("batch:%d", number))
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			var b types.Batch
			if err := json.Unmarshal(val, &b); err != nil {
				return err
			}
			batch = &b
			return nil
		})
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("batch %d not found", number)
		}
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	// Update cache
	bs.mu.Lock()
	bs.batches[number] = batch
	bs.mu.Unlock()

	return batch, nil
}

// GetLatestBatch returns the latest batch
func (bs *BatchStore) GetLatestBatch() (*types.Batch, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if bs.count == 0 {
		return nil, fmt.Errorf("no batches available")
	}

	batch, ok := bs.batches[bs.count]
	if !ok {
		return nil, fmt.Errorf("latest batch not found")
	}

	return batch, nil
}

// GetIncompleteBatch returns the latest batch if it's incomplete (status "pending" and not full)
func (bs *BatchStore) GetIncompleteBatch(batchSize int) (*types.Batch, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	if bs.count == 0 {
		return nil, fmt.Errorf("no batches available")
	}

	batch, ok := bs.batches[bs.count]
	if !ok {
		return nil, fmt.Errorf("latest batch not found")
	}

	// Return batch if it's pending and not full
	if batch.Status == "pending" && len(batch.Transactions) < batchSize {
		return batch, nil
	}

	return nil, fmt.Errorf("no incomplete batch")
}

// GetAllBatches returns all batches
func (bs *BatchStore) GetAllBatches() []*types.Batch {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	batches := make([]*types.Batch, 0, len(bs.batches))
	for i := uint64(1); i <= bs.count; i++ {
		if batch, ok := bs.batches[i]; ok {
			batches = append(batches, batch)
		}
	}

	return batches
}

// Count returns the number of batches
func (bs *BatchStore) Count() uint64 {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.count
}

// loadBatches loads batches from BadgerDB
func (bs *BatchStore) loadBatches() error {
	return bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("batch:")
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var batch types.Batch
				if err := json.Unmarshal(val, &batch); err != nil {
					return err
				}

				bs.batches[batch.Number] = &batch
				if batch.Number > bs.count {
					bs.count = batch.Number
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetLastProcessedBlock returns the last processed block number
func (bs *BatchStore) GetLastProcessedBlock() (uint64, error) {
	var lastBlock uint64
	err := bs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("state:lastBlock"))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				// No last block stored, return 0
				lastBlock = 0
				return nil
			}
			return err
		}

		return item.Value(func(val []byte) error {
			// Parse as uint64 from bytes (little-endian)
			if len(val) >= 8 {
				lastBlock = binary.LittleEndian.Uint64(val)
			}
			return nil
		})
	})

	return lastBlock, err
}

// SetLastProcessedBlock saves the last processed block number
func (bs *BatchStore) SetLastProcessedBlock(blockNum uint64) error {
	// Convert uint64 to 8 bytes (little-endian)
	val := make([]byte, 8)
	binary.LittleEndian.PutUint64(val, blockNum)

	return bs.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("state:lastBlock"), val)
	})
}

// Close closes the BadgerDB connection
func (bs *BatchStore) Close() error {
	return bs.db.Close()
}
