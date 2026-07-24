package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/nowafinance/nowa-zk/indexer/internal/indexer/types"
	"github.com/nowafinance/nowa-zk/indexer/pkg/rpc"
)

// BatchBuilder builds batches from transactions
type BatchBuilder struct {
	batchNum     uint64
	rpcClient    *rpc.Client
	ctx          context.Context
	processedTxs map[string]bool
	mu           sync.Mutex
}

// NewBatchBuilder creates a new batch builder
func NewBatchBuilder(initialBatchNum uint64, initialStateRoot string, rpcClient *rpc.Client, ctx context.Context) *BatchBuilder {
	return &BatchBuilder{
		batchNum:     initialBatchNum,
		rpcClient:    rpcClient,
		ctx:          ctx,
		processedTxs: make(map[string]bool),
	}
}

// filterProcessedTxs filters out transactions that have already been processed
func (bb *BatchBuilder) filterProcessedTxs(txs []*rpc.Transaction) []*rpc.Transaction {
	var uniqueTxs []*rpc.Transaction
	for _, tx := range txs {
		if bb.processedTxs[tx.Hash] {
			fmt.Printf("⚠️  Skipping duplicate transaction %s (already processed)\n", tx.Hash)
			continue
		}
		uniqueTxs = append(uniqueTxs, tx)
	}
	return uniqueTxs
}

// BuildBatch creates a new batch directly from provided transactions
func (bb *BatchBuilder) BuildBatch(txs []*rpc.Transaction, blockNumber uint64) (*types.Batch, error) {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	txs = bb.filterProcessedTxs(txs)
	if len(txs) == 0 {
		return nil, fmt.Errorf("no transactions provided (or all were duplicates)")
	}

	traces := make([]*types.ExecutionTrace, len(txs))

	for i, tx := range txs {
		trace := &types.ExecutionTrace{
			TxHash:               tx.Hash,
			From:                 tx.From,
			Value:                tx.Value.String(),
			Nonce:                tx.Nonce,
			IsContractDeployment: tx.IsContractDeployment,
			OldBalance:           "0",
			NewBalance:           "0",
		}

		if tx.IsContractDeployment {
			trace.ContractAddress = tx.ContractAddress
		} else {
			trace.To = tx.To
		}
		traces[i] = trace
		
		bb.processedTxs[tx.Hash] = true
	}

	batch := &types.Batch{
		Number:       bb.batchNum,
		Transactions: txs,
		OldStateRoot: "0x0",
		NewStateRoot: "0x0",
		Timestamp:    time.Now().Unix(),
		Status:       "pending",
		Traces:       traces,
	}

	var allParsedTrades []*types.ParsedTrade
	for _, tx := range txs {
		if trades, err := ParseTradesFromTxData(tx.Data); err == nil && len(trades) > 0 {
			allParsedTrades = append(allParsedTrades, trades...)
		} else if err != nil {
			fmt.Printf("⚠️  Failed to parse trades from tx %s: %v\n", tx.Hash, err)
		}
	}
	batch.Trades = allParsedTrades
	batch.Hash = bb.computeBatchHash(batch)

	bb.batchNum++
	return batch, nil
}

// AppendToBatch appends transactions to an existing batch
func (bb *BatchBuilder) AppendToBatch(batch *types.Batch, newTxs []*rpc.Transaction, blockNumber uint64) error {
	bb.mu.Lock()
	defer bb.mu.Unlock()

	newTxs = bb.filterProcessedTxs(newTxs)
	if len(newTxs) == 0 {
		return fmt.Errorf("no transactions to append (or all were duplicates)")
	}

	newTraces := make([]*types.ExecutionTrace, len(newTxs))

	for i, tx := range newTxs {
		trace := &types.ExecutionTrace{
			TxHash:               tx.Hash,
			From:                 tx.From,
			Value:                tx.Value.String(),
			Nonce:                tx.Nonce,
			IsContractDeployment: tx.IsContractDeployment,
			OldBalance:           "0",
			NewBalance:           "0",
		}

		if tx.IsContractDeployment {
			trace.ContractAddress = tx.ContractAddress
		} else {
			trace.To = tx.To
		}
		newTraces[i] = trace
		
		bb.processedTxs[tx.Hash] = true
	}

	batch.Transactions = append(batch.Transactions, newTxs...)
	batch.Traces = append(batch.Traces, newTraces...)

	for _, tx := range newTxs {
		if trades, err := ParseTradesFromTxData(tx.Data); err == nil && len(trades) > 0 {
			batch.Trades = append(batch.Trades, trades...)
		} else if err != nil {
			fmt.Printf("⚠️  Failed to parse trades from tx %s: %v\n", tx.Hash, err)
		}
	}

	batch.Hash = bb.computeBatchHash(batch)
	return nil
}

// computeBatchHash computes the batch hash
func (bb *BatchBuilder) computeBatchHash(batch *types.Batch) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d:%s:%s:%d",
		batch.Number,
		batch.OldStateRoot,
		batch.NewStateRoot,
		batch.Timestamp))
	for _, tx := range batch.Transactions {
		sb.WriteString(":")
		sb.WriteString(tx.Hash)
	}
	hash := crypto.Keccak256Hash([]byte(sb.String()))
	return hash.Hex()
}
