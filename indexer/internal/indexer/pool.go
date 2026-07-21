package indexer

import (
	"sync"

	"github.com/nowafinance/nowa-zk/indexer/pkg/rpc"
)

// TransactionPool manages pending transactions
type TransactionPool struct {
	mu   sync.RWMutex
	txs  []*rpc.Transaction
	max  int
}

// NewTransactionPool creates a new transaction pool
func NewTransactionPool() *TransactionPool {
	return &TransactionPool{
		txs: make([]*rpc.Transaction, 0),
		max: 10000, // Maximum pool size
	}
}

// AddTransaction adds a transaction to the pool
func (p *TransactionPool) AddTransaction(tx *rpc.Transaction) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if transaction already exists
	for _, existing := range p.txs {
		if existing.Hash == tx.Hash {
			return // Already in pool
		}
	}

	// Add transaction
	p.txs = append(p.txs, tx)

	// Evict oldest if pool is full
	if len(p.txs) > p.max {
		p.txs = p.txs[1:]
	}
}

// GetTransactions returns up to N transactions and removes them from pool
func (p *TransactionPool) GetTransactions(n int) []*rpc.Transaction {
	p.mu.Lock()
	defer p.mu.Unlock()

	if n > len(p.txs) {
		n = len(p.txs)
	}

	txs := make([]*rpc.Transaction, n)
	copy(txs, p.txs[:n])

	// Remove from pool
	p.txs = p.txs[n:]

	return txs
}

// Size returns the current pool size
func (p *TransactionPool) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.txs)
}

// Clear removes all transactions from the pool
func (p *TransactionPool) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.txs = p.txs[:0]
}

