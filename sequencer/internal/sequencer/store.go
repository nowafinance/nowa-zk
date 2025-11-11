package sequencer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// BatchStore stores batches locally
type BatchStore struct {
	mu      sync.RWMutex
	path    string
	batches map[uint64]*Batch
	count   uint64
}

// NewBatchStore creates a new batch store
func NewBatchStore(path string) *BatchStore {
	// Create directory if it doesn't exist
	os.MkdirAll(path, 0755)

	store := &BatchStore{
		path:    path,
		batches: make(map[uint64]*Batch),
	}

	// Load existing batches
	store.loadBatches()

	return store
}

// SaveBatch saves a batch to disk
func (bs *BatchStore) SaveBatch(batch *Batch) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	// Store in memory
	bs.batches[batch.Number] = batch
	if batch.Number > bs.count {
		bs.count = batch.Number
	}

	// Save to disk
	filename := filepath.Join(bs.path, fmt.Sprintf("batch_%d.json", batch.Number))
	data, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write batch file: %w", err)
	}

	return nil
}

// GetBatch retrieves a batch by number
func (bs *BatchStore) GetBatch(number uint64) (*Batch, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	batch, ok := bs.batches[number]
	if !ok {
		return nil, fmt.Errorf("batch %d not found", number)
	}

	return batch, nil
}

// GetLatestBatch returns the latest batch
func (bs *BatchStore) GetLatestBatch() (*Batch, error) {
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

// GetAllBatches returns all batches
func (bs *BatchStore) GetAllBatches() []*Batch {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	batches := make([]*Batch, 0, len(bs.batches))
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

// loadBatches loads batches from disk
func (bs *BatchStore) loadBatches() {
	files, err := filepath.Glob(filepath.Join(bs.path, "batch_*.json"))
	if err != nil {
		return
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var batch Batch
		if err := json.Unmarshal(data, &batch); err != nil {
			continue
		}

		bs.batches[batch.Number] = &batch
		if batch.Number > bs.count {
			bs.count = batch.Number
		}
	}
}

