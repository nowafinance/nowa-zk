package sequencer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/tannetwork/zk-sequencer/sequencer/internal/sequencer/types"
	"github.com/tannetwork/zk-sequencer/sequencer/pkg/rpc"
)

// BatchBuilder builds batches from transactions
type BatchBuilder struct {
	batchNum  uint64
	stateRoot string
	mu        sync.Mutex
}

// NewBatchBuilder creates a new batch builder
func NewBatchBuilder(initialBatchNum uint64, initialStateRoot string) *BatchBuilder {
	return &BatchBuilder{
		batchNum:  initialBatchNum,
		stateRoot: initialStateRoot,
	}
}

// BuildBatch creates a new batch directly from provided transactions
func (bb *BatchBuilder) BuildBatch(txs []*rpc.Transaction) (*types.Batch, error) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	if len(txs) == 0 {
		return nil, fmt.Errorf("no transactions provided")
	}

	// Generate execution traces (simplified for now)
	traces := make([]*types.ExecutionTrace, len(txs))
	for i, tx := range txs {
		trace := &types.ExecutionTrace{
			TxHash:               tx.Hash,
			From:                 tx.From,
			Value:                tx.Value.String(),
			Nonce:                tx.Nonce,
			IsContractDeployment: tx.IsContractDeployment,
		}

		// For contract deployments, use contract address instead of to
		if tx.IsContractDeployment {
			trace.ContractAddress = tx.ContractAddress
		} else {
			trace.To = tx.To
		}

		traces[i] = trace
	}

	// Compute new state root (simplified - just hash of batch)
	oldRoot := bb.stateRoot
	newRoot := bb.computeStateRoot(txs, oldRoot)

	// Create batch
	batch := &types.Batch{
		Number:       bb.batchNum,
		Transactions: txs,
		OldStateRoot: oldRoot,
		NewStateRoot: newRoot,
		Timestamp:    time.Now().Unix(),
		Status:       "pending",
		Traces:       traces,
	}

	// Compute batch hash
	batch.Hash = bb.computeBatchHash(batch)

	// Update state
	bb.stateRoot = newRoot
	bb.batchNum++

	return batch, nil
}

// AppendToBatch appends transactions to an existing batch and updates state root
func (bb *BatchBuilder) AppendToBatch(batch *types.Batch, newTxs []*rpc.Transaction) error {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	if len(newTxs) == 0 {
		return fmt.Errorf("no transactions to append")
	}

	// Generate execution traces for new transactions
	newTraces := make([]*types.ExecutionTrace, len(newTxs))
	for i, tx := range newTxs {
		trace := &types.ExecutionTrace{
			TxHash:               tx.Hash,
			From:                 tx.From,
			Value:                tx.Value.String(),
			Nonce:                tx.Nonce,
			IsContractDeployment: tx.IsContractDeployment,
		}

		// For contract deployments, use contract address instead of to
		if tx.IsContractDeployment {
			trace.ContractAddress = tx.ContractAddress
		} else {
			trace.To = tx.To
		}

		newTraces[i] = trace
	}

	// Append transactions and traces
	batch.Transactions = append(batch.Transactions, newTxs...)
	batch.Traces = append(batch.Traces, newTraces...)

	// Recompute state root with all transactions
	newRoot := bb.computeStateRoot(batch.Transactions, batch.OldStateRoot)
	batch.NewStateRoot = newRoot

	// Recompute batch hash (since transactions changed)
	batch.Hash = bb.computeBatchHash(batch)

	// Update builder's state root to match
	bb.stateRoot = newRoot

	return nil
}

// computeStateRoot computes a new state root (simplified implementation)
func (bb *BatchBuilder) computeStateRoot(txs []*rpc.Transaction, oldRoot string) string {
	// Simplified: hash of old root + all transaction hashes
	data := oldRoot
	for _, tx := range txs {
		data += tx.Hash
	}
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])
}

// computeBatchHash computes the batch hash
func (bb *BatchBuilder) computeBatchHash(batch *types.Batch) string {
	data := fmt.Sprintf("%d:%s:%s:%d",
		batch.Number,
		batch.OldStateRoot,
		batch.NewStateRoot,
		batch.Timestamp)
	for _, tx := range batch.Transactions {
		data += ":" + tx.Hash
	}
	hash := sha256.Sum256([]byte(data))
	return "0x" + hex.EncodeToString(hash[:])
}
