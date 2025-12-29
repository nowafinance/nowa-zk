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
// ProofData stores metadata about a proven batch (no proof/witness, just hashes)
type ProofData struct {
	BatchNumber  uint64   `json:"batch_number"`
	BatchHash    string   `json:"batch_hash,omitempty"`
	TxHash       string   `json:"tx_hash,omitempty"`
	TxHashes     []string `json:"tx_hashes,omitempty"`
	Timestamp    int64    `json:"timestamp"`
	Submitter    string   `json:"submitter,omitempty"`
	NewStateRoot string   `json:"new_state_root,omitempty"`
	Status       uint8    `json:"status,omitempty"`
}

// SaveMetadata saves metadata for a proven batch (batch hash, L1 tx hash + L2 tx hashes)
func (s *ProverStore) SaveMetadata(batchNumber uint64, batchHash, txHash string, txHashes []string, submitter, newStateRoot string, status uint8) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := ProofData{
		BatchNumber:  batchNumber,
		BatchHash:    batchHash,
		TxHash:       txHash,
		TxHashes:     txHashes,
		Timestamp:    time.Now().Unix(),
		Submitter:    submitter,
		NewStateRoot: newStateRoot,
		Status:       status,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal proof data: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte(fmt.Sprintf("proof_%d", batchNumber))
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

// GetBatches retrieves multiple batches with pagination in descending order (newest first)
// offset: number of batches to skip from newest, limit: max batches to return
func (s *ProverStore) GetBatches(offset, limit uint64) ([]*ProofData, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var batches []*ProofData

	// First, collect all batch numbers
	var batchNumbers []uint64
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("proof_")
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			var batchNum uint64
			_, err := fmt.Sscanf(string(item.Key()), "proof_%d", &batchNum)
			if err != nil {
				continue
			}
			batchNumbers = append(batchNumbers, batchNum)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort in descending order (newest first)
	// Simple bubble sort for descending order
	for i := 0; i < len(batchNumbers)-1; i++ {
		for j := 0; j < len(batchNumbers)-i-1; j++ {
			if batchNumbers[j] < batchNumbers[j+1] {
				batchNumbers[j], batchNumbers[j+1] = batchNumbers[j+1], batchNumbers[j]
			}
		}
	}

	// Apply offset and limit
	start := int(offset)
	end := start + int(limit)
	if start >= len(batchNumbers) {
		return batches, nil // Return empty if offset too large
	}
	if end > len(batchNumbers) {
		end = len(batchNumbers)
	}

	selectedBatches := batchNumbers[start:end]

	// Fetch the actual batch data
	err = s.db.View(func(txn *badger.Txn) error {
		for _, batchNum := range selectedBatches {
			key := []byte(fmt.Sprintf("proof_%d", batchNum))
			item, err := txn.Get(key)
			if err != nil {
				continue
			}

			err = item.Value(func(val []byte) error {
				var data ProofData
				if err := json.Unmarshal(val, &data); err != nil {
					return err
				}
				batches = append(batches, &data)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return batches, nil
}

// GetTotalBatches returns the total number of proven batches stored
func (s *ProverStore) GetTotalBatches() (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count uint64
	err := s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("proof_")
		opts.PrefetchValues = false // Key-only iteration is enough
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			count++
		}
		return nil
	})

	return count, err
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

// VerificationFailure represents a failed verification attempt
type VerificationFailure struct {
	BatchNumber uint64 `json:"batch_number"`
	ErrorMsg    string `json:"error_msg"`
	ProofData   []byte `json:"proof_data,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}

// SaveVerificationFailure saves details about a failed verification
func (s *ProverStore) SaveVerificationFailure(batchNumber uint64, errorMsg string, proofData []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	failure := VerificationFailure{
		BatchNumber: batchNumber,
		ErrorMsg:    errorMsg,
		ProofData:   proofData,
		Timestamp:   time.Now().Unix(),
	}

	jsonData, err := json.Marshal(failure)
	if err != nil {
		return fmt.Errorf("failed to marshal failure data: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		key := []byte(fmt.Sprintf("failure_%d", batchNumber))
		if err := txn.Set(key, jsonData); err != nil {
			return err
		}
		// Also save as "last_failure" for quick access
		return txn.Set([]byte("last_failure"), jsonData)
	})
}

// GetLastVerificationFailure retrieves the last failure if any
func (s *ProverStore) GetLastVerificationFailure() (uint64, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var failure VerificationFailure
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("last_failure"))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &failure)
		})
	})

	if err != nil {
		return 0, "", err
	}
	return failure.BatchNumber, failure.ErrorMsg, nil
}

// GetVerificationFailure retrieves a specific verification failure
func (s *ProverStore) GetVerificationFailure(batchNumber uint64) (*VerificationFailure, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var failure VerificationFailure
	err := s.db.View(func(txn *badger.Txn) error {
		key := []byte(fmt.Sprintf("failure_%d", batchNumber))
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &failure)
		})
	})

	if err != nil {
		return nil, err
	}
	return &failure, nil
}

// HaltState represents the prover halt state
type HaltState struct {
	Halted    bool   `json:"halted"`
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}

// SetHaltState marks the prover as halted
func (s *ProverStore) SetHaltState(reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := HaltState{
		Halted:    true,
		Reason:    reason,
		Timestamp: time.Now().Unix(),
	}

	jsonData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal halt state: %w", err)
	}

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("halt_state"), jsonData)
	})
}

// GetHaltState checks if prover is in halt state
func (s *ProverStore) GetHaltState() (bool, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var state HaltState
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("halt_state"))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				// No halt state means not halted
				state.Halted = false
				return nil
			}
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &state)
		})
	})

	if err != nil {
		return false, "", err
	}
	return state.Halted, state.Reason, nil
}

// ClearHaltState clears the halt flag (for manual recovery)
func (s *ProverStore) ClearHaltState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte("halt_state"))
	})
}
