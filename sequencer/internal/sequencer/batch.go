package sequencer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/tannetwork/zk-sequencer/sequencer/pkg/rpc"
)

// Batch represents a batch of transactions
type Batch struct {
	Number       uint64                `json:"number"`
	Hash         string                `json:"hash"`
	Transactions []*rpc.Transaction    `json:"transactions"`
	OldStateRoot string                `json:"oldStateRoot"`
	NewStateRoot string                `json:"newStateRoot"`
	Timestamp    int64                 `json:"timestamp"`
	Status       string                `json:"status"` // "pending", "proving", "ready", "submitted"
	Traces       []*ExecutionTrace     `json:"traces,omitempty"`
}

// ExecutionTrace represents execution trace for a transaction
type ExecutionTrace struct {
	TxHash      string `json:"txHash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	Nonce       uint64 `json:"nonce"`
	OldBalance  string `json:"oldBalance,omitempty"`
	NewBalance  string `json:"newBalance,omitempty"`
}

// BatchBuilder builds batches from transactions
type BatchBuilder struct {
	txPool     *TransactionPool
	batchSize  int
	batchNum   uint64
	stateRoot  string
	mu         sync.Mutex
}

// NewBatchBuilder creates a new batch builder
func NewBatchBuilder(txPool *TransactionPool, batchSize int) *BatchBuilder {
	return &BatchBuilder{
		txPool:    txPool,
		batchSize: batchSize,
		batchNum:  1,
		stateRoot: "0x0000000000000000000000000000000000000000000000000000000000000000",
	}
}

// BuildBatch creates a new batch from the transaction pool
func (bb *BatchBuilder) BuildBatch() (*Batch, error) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	// Get transactions from pool
	txs := bb.txPool.GetTransactions(bb.batchSize)
	if len(txs) == 0 {
		return nil, fmt.Errorf("no transactions available")
	}

	// Generate execution traces (simplified for now)
	traces := make([]*ExecutionTrace, len(txs))
	for i, tx := range txs {
		traces[i] = &ExecutionTrace{
			TxHash: tx.Hash,
			From:   tx.From,
			To:     tx.To,
			Value:  tx.Value.String(),
			Nonce:  tx.Nonce,
		}
	}

	// Compute new state root (simplified - just hash of batch)
	oldRoot := bb.stateRoot
	newRoot := bb.computeStateRoot(txs, oldRoot)

	// Create batch
	batch := &Batch{
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
func (bb *BatchBuilder) computeBatchHash(batch *Batch) string {
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

