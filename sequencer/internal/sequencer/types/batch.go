package types

import "github.com/tannetwork/zk-sequencer/sequencer/pkg/rpc"

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
	TxHash            string `json:"txHash"`
	From              string `json:"from"`
	To                string `json:"to,omitempty"`                // Empty for contract deployments
	ContractAddress   string `json:"contractAddress,omitempty"`   // Set for contract deployments
	IsContractDeployment bool `json:"isContractDeployment"`      // True if this is a contract deployment
	Value             string `json:"value"`
	Nonce             uint64 `json:"nonce"`
	OldBalance        string `json:"oldBalance,omitempty"`
	NewBalance        string `json:"newBalance,omitempty"`
}

