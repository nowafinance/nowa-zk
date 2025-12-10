package sequencer

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/tannetwork/zk-sequencer/sequencer/internal/sequencer/types"
	"github.com/tannetwork/zk-sequencer/sequencer/pkg/rpc"
	"github.com/tannetwork/zk-sequencer/sequencer/pkg/smt"
)

// BatchBuilder builds batches from transactions
type BatchBuilder struct {
	batchNum  uint64
	stateRoot string
	smt       *smt.SparseMerkleTree // Sparse Merkle Tree for state root calculation
	rpcClient *rpc.Client           // RPC client for querying balances
	ctx       context.Context       // Context for RPC calls
	mu        sync.Mutex
}

// NewBatchBuilder creates a new batch builder
func NewBatchBuilder(initialBatchNum uint64, initialStateRoot string, rpcClient *rpc.Client, ctx context.Context) *BatchBuilder {
	return &BatchBuilder{
		batchNum:  initialBatchNum,
		stateRoot: initialStateRoot,
		smt:       smt.NewSparseMerkleTree(),
		rpcClient: rpcClient,
		ctx:       ctx,
	}
}

// filterProcessedTxs filters out transactions that have already been processed
func (bb *BatchBuilder) filterProcessedTxs(txs []*rpc.Transaction) []*rpc.Transaction {
	var uniqueTxs []*rpc.Transaction
	for _, tx := range txs {
		txKey := fmt.Sprintf("tx:%s", tx.Hash)
		// Check if transaction hash already exists in SMT
		if _, exists := bb.smt.Get(txKey); exists {
			// Already processed, skip
			fmt.Printf("⚠️  Skipping duplicate transaction %s (already processed)\n", tx.Hash)
			continue
		}
		uniqueTxs = append(uniqueTxs, tx)
	}
	return uniqueTxs
}

// BuildBatch creates a new batch directly from provided transactions
// blockNumber is used to query balances at the start of the block
func (bb *BatchBuilder) BuildBatch(txs []*rpc.Transaction, blockNumber uint64) (*types.Batch, error) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	// Filter out already processed transactions
	txs = bb.filterProcessedTxs(txs)

	if len(txs) == 0 {
		return nil, fmt.Errorf("no transactions provided (or all were duplicates)")
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

	// Compute new state root using Sparse Merkle Tree
	oldRoot := bb.stateRoot

	// Update SMT with transaction state changes
	// Query balances at block start (blockNumber - 1, or blockNumber if first tx in block)
	blockNumForBalance := blockNumber
	if blockNumber > 0 {
		blockNumForBalance = blockNumber - 1 // Balance at start of block
	}

	for _, tx := range txs {
		if err := bb.updateStateFromTransaction(tx, blockNumForBalance); err != nil {
			return nil, fmt.Errorf("failed to update state from transaction: %w", err)
		}
	}

	// Get new root from SMT
	newRoot := bb.smt.RootHex()

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
// blockNumber is used to query balances at the start of the block
func (bb *BatchBuilder) AppendToBatch(batch *types.Batch, newTxs []*rpc.Transaction, blockNumber uint64) error {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	// Filter out already processed transactions
	newTxs = bb.filterProcessedTxs(newTxs)

	if len(newTxs) == 0 {
		return fmt.Errorf("no transactions to append (or all were duplicates)")
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

	// Update SMT with new transaction state changes
	blockNumForBalance := blockNumber
	if blockNumber > 0 {
		blockNumForBalance = blockNumber - 1 // Balance at start of block
	}

	for _, tx := range newTxs {
		if err := bb.updateStateFromTransaction(tx, blockNumForBalance); err != nil {
			return fmt.Errorf("failed to update state from transaction: %w", err)
		}
	}

	// Get new root from SMT
	newRoot := bb.smt.RootHex()
	batch.NewStateRoot = newRoot

	// Recompute batch hash (since transactions changed)
	batch.Hash = bb.computeBatchHash(batch)

	// NOTE: Do NOT update bb.stateRoot here!
	// The builder's state root should only advance when a batch is completed.
	// Otherwise, incomplete batches corrupt the state for future batches.

	return nil
}

// updateStateFromTransaction updates the SMT with state changes from a transaction
func (bb *BatchBuilder) updateStateFromTransaction(tx *rpc.Transaction, blockNumber uint64) error {
	// Helper to get or query balance
	getBalance := func(address string) (*big.Int, error) {
		balanceKey := fmt.Sprintf("balance:%s", address)

		// Check if balance exists in SMT
		if balanceBytes, exists := bb.smt.Get(balanceKey); exists {
			// Parse balance from bytes
			balance := new(big.Int)
			balance.SetBytes(balanceBytes)
			return balance, nil
		}

		// Query balance from blockchain at block start
		balance, err := bb.rpcClient.GetBalanceAtBlock(bb.ctx, address, blockNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to get balance for %s: %w", address, err)
		}

		// Store in SMT
		bb.smt.Update(balanceKey, balance.Bytes())
		return balance, nil
	}

	// Update sender balance
	senderBalance, err := getBalance(tx.From)
	if err != nil {
		return err
	}

	// Validate sender has sufficient balance
	totalDebit := new(big.Int).Add(tx.Value, big.NewInt(0)) // In future, add gas fees here
	if senderBalance.Cmp(totalDebit) < 0 {
		return fmt.Errorf("insufficient balance: account %s has %s wei, needs %s wei",
			tx.From, senderBalance.String(), totalDebit.String())
	}

	// Decrease sender balance
	newSenderBalance := new(big.Int).Sub(senderBalance, tx.Value)
	if newSenderBalance.Sign() < 0 {
		// This should never happen due to the check above, but add as safety check
		return fmt.Errorf("internal error: negative balance after transfer for %s", tx.From)
	}
	senderKey := fmt.Sprintf("balance:%s", tx.From)
	bb.smt.Update(senderKey, newSenderBalance.Bytes())

	// Update receiver balance (if not contract deployment)
	if !tx.IsContractDeployment && tx.To != "" {
		receiverBalance, err := getBalance(tx.To)
		if err != nil {
			return err
		}

		// Increase receiver balance
		newReceiverBalance := new(big.Int).Add(receiverBalance, tx.Value)
		receiverKey := fmt.Sprintf("balance:%s", tx.To)
		bb.smt.Update(receiverKey, newReceiverBalance.Bytes())
	}

	// Store transaction hash for tracking (optional)
	txKey := fmt.Sprintf("tx:%s", tx.Hash)
	bb.smt.Update(txKey, []byte(tx.Hash))

	return nil
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
	hash := crypto.Keccak256Hash([]byte(data))
	return hash.Hex()
}
