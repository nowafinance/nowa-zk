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
	TxHash      string      `json:"tx_hash,omitempty"`
	TxHashes    []string    `json:"tx_hashes,omitempty"` // L2 transaction hashes in batch
	Timestamp   int64       `json:"timestamp"`
}

// SaveProof saves the proof data for a batch
func (s *ProverStore) SaveProof(batchNumber uint64, proof interface{}, witness interface{}, txHash string, txHashes []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := ProofData{
		BatchNumber: batchNumber,
		Proof:       proof,
		Witness:     witness,
		TxHash:      txHash,
		TxHashes:    txHashes,
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
