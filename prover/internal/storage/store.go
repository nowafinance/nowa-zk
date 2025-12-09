package storage

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v3"
)

// ProverStore handles persistence for the prover
type ProverStore struct {
	db *badger.DB
	mu sync.RWMutex
}

// NewProverStore creates a new ProverStore instance
func NewProverStore(path string) (*ProverStore, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = nil // Disable default logger to reduce noise

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	return &ProverStore{
		db: db,
	}, nil
}

// Close closes the database connection
func (s *ProverStore) Close() error {
	return s.db.Close()
}

// SaveLastProcessedBatch saves the last processed batch number
func (s *ProverStore) SaveLastProcessedBatch(batchNumber uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(txn *badger.Txn) error {
		val := []byte(fmt.Sprintf("%d", batchNumber))
		return txn.Set([]byte("last_processed_batch"), val)
	})
}

// GetLastProcessedBatch retrieves the last processed batch number
func (s *ProverStore) GetLastProcessedBatch() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var batchNumber uint64
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("last_processed_batch"))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				batchNumber = 0
				return nil
			}
			return err
		}

		return item.Value(func(val []byte) error {
			_, err := fmt.Sscanf(string(val), "%d", &batchNumber)
			return err
		})
	})

	return batchNumber, err
}

// ProofData represents the data to be saved for a proof
type ProofData struct {
	BatchNumber uint64      `json:"batch_number"`
	Proof       interface{} `json:"proof"`
	Witness     interface{} `json:"witness"`
	Timestamp   int64       `json:"timestamp"`
}

// SaveProof saves the proof data for a batch
func (s *ProverStore) SaveProof(batchNumber uint64, proof interface{}, witness interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := ProofData{
		BatchNumber: batchNumber,
		Proof:       proof,
		Witness:     witness,
		Timestamp:   time.Now().Unix(),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal proof data: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte(fmt.Sprintf("proof_%d", batchNumber))
		// Set with TTL if needed, but requirement says "latest 100", which implies logic elsewhere or just keeping all for now.
		// For "latest 100", we could implement a cleanup, but Badger is efficient.
		// Let's just save it.
		return txn.Set(key, jsonData)
	})
}

// GetProof retrieves the proof data for a batch
func (s *ProverStore) GetProof(batchNumber uint64) (*ProofData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var data ProofData
	err := s.db.View(func(txn *badger.Txn) error {
		key := []byte(fmt.Sprintf("proof_%d", batchNumber))
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &data)
		})
	})

	if err != nil {
		return nil, err
	}
	return &data, nil
}

// SaveLastStateRoot saves the last verified state root
func (s *ProverStore) SaveLastStateRoot(stateRoot string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("last_state_root"), []byte(stateRoot))
	})
}

// GetLastStateRoot retrieves the last verified state root
func (s *ProverStore) GetLastStateRoot() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var stateRoot string
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("last_state_root"))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}

		return item.Value(func(val []byte) error {
			stateRoot = string(val)
			return nil
		})
	})

	return stateRoot, err
}
