package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
)

// Transaction represents an Ethereum transaction
type Transaction struct {
	Hash        string   `json:"hash"`
	From        string   `json:"from"`
	To          string   `json:"to"`
	Value       *big.Int `json:"value"`
	Data        string   `json:"input"`
	Nonce       uint64   `json:"nonce"`
	Gas         uint64   `json:"gas"`
	GasPrice    *big.Int `json:"gasPrice"`
	GasTipCap   *big.Int `json:"maxFeePerGas,omitempty"` // EIP-1559
	GasFeeCap   *big.Int `json:"maxPriorityFeePerGas,omitempty"` // EIP-1559
	BlockNumber *big.Int `json:"blockNumber,omitempty"`
	BlockHash   string   `json:"blockHash,omitempty"`
	Index       uint64   `json:"transactionIndex,omitempty"`
}

// GetTransactionByHash fetches a transaction by hash
func (c *Client) GetTransactionByHash(ctx context.Context, txHash string) (*Transaction, error) {
	params := []interface{}{txHash}

	resp, err := c.call(ctx, "eth_getTransactionByHash", params)
	if err != nil {
		return nil, err
	}

	var tx Transaction
	if err := json.Unmarshal(resp.Result, &tx); err != nil {
		// Try parsing as raw map first
		var raw map[string]interface{}
		if err := json.Unmarshal(resp.Result, &raw); err != nil {
			return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
		}
		tx = parseTransaction(raw)
	}

	return &tx, nil
}

// GetTransactionsByBlock fetches all transactions from a block
func (c *Client) GetTransactionsByBlock(ctx context.Context, blockNumber uint64) ([]*Transaction, error) {
	block, err := c.GetBlockByNumber(ctx, blockNumber, true)
	if err != nil {
		return nil, err
	}

	return block.Transactions, nil
}

// parseTransaction parses a transaction from raw JSON map
func parseTransaction(raw map[string]interface{}) Transaction {
	tx := Transaction{}

	if hash, ok := raw["hash"].(string); ok {
		tx.Hash = hash
	}
	if from, ok := raw["from"].(string); ok {
		tx.From = from
	}
	if to, ok := raw["to"].(string); ok {
		tx.To = to
	}
	if value, ok := raw["value"].(string); ok {
		tx.Value = parseHexBigInt(value)
	}
	if input, ok := raw["input"].(string); ok {
		tx.Data = input
	}
	if nonce, ok := raw["nonce"].(string); ok {
		tx.Nonce = parseHexBigInt(nonce).Uint64()
	}
	if gas, ok := raw["gas"].(string); ok {
		tx.Gas = parseHexBigInt(gas).Uint64()
	}
	if gasPrice, ok := raw["gasPrice"].(string); ok {
		tx.GasPrice = parseHexBigInt(gasPrice)
	}
	// EIP-1559 fields
	if maxFeePerGas, ok := raw["maxFeePerGas"].(string); ok {
		tx.GasTipCap = parseHexBigInt(maxFeePerGas)
	}
	if maxPriorityFeePerGas, ok := raw["maxPriorityFeePerGas"].(string); ok {
		tx.GasFeeCap = parseHexBigInt(maxPriorityFeePerGas)
	}
	if blockNumber, ok := raw["blockNumber"].(string); ok {
		tx.BlockNumber = parseHexBigInt(blockNumber)
	}
	if blockHash, ok := raw["blockHash"].(string); ok {
		tx.BlockHash = blockHash
	}
	if txIndex, ok := raw["transactionIndex"].(string); ok {
		tx.Index = parseHexBigInt(txIndex).Uint64()
	}

	return tx
}

