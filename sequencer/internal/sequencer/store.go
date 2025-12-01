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
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	// Open BadgerDB (use subdirectory for DB files)
	dbPath := filepath.Join(path, "db")
	fmt.Printf("📂 Opening BadgerDB at %s\n", dbPath)
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
	count := 0
	err := bs.db.View(func(txn *badger.Txn) error {
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
				count++
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	fmt.Printf("✅ Loaded %d batches from DB. Max batch number: %d\n", count, bs.count)
	return err
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

// SetBlockHash saves the hash for a block number (for reorg detection)
func (bs *BatchStore) SetBlockHash(blockNum uint64, blockHash string) error {
	key := []byte(fmt.Sprintf("blockhash:%d", blockNum))
	return bs.db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, []byte(blockHash))
	})
}

// GetBlockHash retrieves the hash for a block number
func (bs *BatchStore) GetBlockHash(blockNum uint64) (string, error) {
	var blockHash string
	err := bs.db.View(func(txn *badger.Txn) error {
		key := []byte(fmt.Sprintf("blockhash:%d", blockNum))
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return fmt.Errorf("block hash not found for block %d", blockNum)
			}
			return err
		}

		return item.Value(func(val []byte) error {
			blockHash = string(val)
			return nil
		})
	})

	return blockHash, err
}

// DeleteBatchesAfter deletes all batches after the specified batch number (for reorg rollback)
func (bs *BatchStore) DeleteBatchesAfter(batchNum uint64) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	return bs.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("batch:")
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()

			// Extract batch number from key "batch:123"
			var batchNumFromKey uint64
			fmt.Sscanf(string(key), "batch:%d", &batchNumFromKey)

			if batchNumFromKey > batchNum {
				// Delete this batch
				if err := txn.Delete(key); err != nil {
					return err
				}
				// Remove from in-memory cache
				delete(bs.batches, batchNumFromKey)
			}
		}

		// Update count
		if batchNum < bs.count {
			bs.count = batchNum
		}

		return nil
	})
}

// ClearAll deletes all batches, block hashes, and state data
func (bs *BatchStore) ClearAll() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	return bs.db.Update(func(txn *badger.Txn) error {
		// Delete all batches
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("batch:")
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			key := it.Item().Key()
			if err := txn.Delete(key); err != nil {
				return err
			}
		}
		it.Close() // Close first iterator explicitly

		// Delete all block hashes
		opts.Prefix = []byte("blockhash:")
		it = txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			key := it.Item().Key()
			if err := txn.Delete(key); err != nil {
				return err
			}
		}

		// Delete last processed block
		if err := txn.Delete([]byte("state:lastBlock")); err != nil && err != badger.ErrKeyNotFound {
			return err
		}

		// Clear in-memory cache
		bs.batches = make(map[uint64]*types.Batch)
		bs.count = 0

		return nil
	})
}

// Close closes the BadgerDB connection
func (bs *BatchStore) Close() error {
	return bs.db.Close()
}
