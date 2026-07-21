package rpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/crypto/sha3"
)

// Transaction represents an Ethereum transaction
type Transaction struct {
	Hash            string   `json:"hash"`
	From            string   `json:"from"`
	To              string   `json:"to"`              // Empty for contract deployments
	ContractAddress string   `json:"contractAddress,omitempty"` // Set for contract deployments
	Value           *big.Int `json:"value"`
	Data            string   `json:"input"`
	Nonce           uint64   `json:"nonce"`
	Gas             uint64   `json:"gas"`
	GasPrice        *big.Int `json:"gasPrice"`
	GasTipCap       *big.Int `json:"maxFeePerGas,omitempty"` // EIP-1559
	GasFeeCap       *big.Int `json:"maxPriorityFeePerGas,omitempty"` // EIP-1559
	BlockNumber     *big.Int `json:"blockNumber,omitempty"`
	BlockHash       string   `json:"blockHash,omitempty"`
	Index           uint64   `json:"transactionIndex,omitempty"`
	IsContractDeployment bool `json:"isContractDeployment"` // True if this is a contract deployment
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

	// Check if this is a contract deployment (to is empty/null and data is not empty)
	if tx.To == "" || tx.To == "0x" || tx.To == "0x0000000000000000000000000000000000000000" {
		if tx.Data != "" && tx.Data != "0x" {
			tx.IsContractDeployment = true
			// Try to get contract address from receipt
			receipt, err := c.GetTransactionReceipt(ctx, txHash)
			if err == nil && receipt != nil && receipt.ContractAddress != "" {
				tx.ContractAddress = receipt.ContractAddress
			} else {
				// Fallback: compute contract address from sender + nonce
				tx.ContractAddress = ComputeContractAddress(tx.From, tx.Nonce)
			}
		}
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

// TransactionReceipt represents a transaction receipt
type TransactionReceipt struct {
	TransactionHash   string   `json:"transactionHash"`
	TransactionIndex  string   `json:"transactionIndex"`
	BlockNumber       string   `json:"blockNumber"`
	BlockHash         string   `json:"blockHash"`
	From              string   `json:"from"`
	To                string   `json:"to"`
	ContractAddress   string   `json:"contractAddress"` // Set for contract deployments
	GasUsed           string   `json:"gasUsed"`
	CumulativeGasUsed string   `json:"cumulativeGasUsed"`
	Status            string   `json:"status"` // "0x1" for success, "0x0" for failure
	Logs              []interface{} `json:"logs"`
}

// GetTransactionReceipt fetches a transaction receipt by hash
func (c *Client) GetTransactionReceipt(ctx context.Context, txHash string) (*TransactionReceipt, error) {
	params := []interface{}{txHash}

	resp, err := c.call(ctx, "eth_getTransactionReceipt", params)
	if err != nil {
		return nil, err
	}

	var receipt TransactionReceipt
	if err := json.Unmarshal(resp.Result, &receipt); err != nil {
		// Try parsing as raw map first
		var raw map[string]interface{}
		if err := json.Unmarshal(resp.Result, &raw); err != nil {
			return nil, fmt.Errorf("failed to unmarshal receipt: %w", err)
		}
		receipt = parseTransactionReceipt(raw)
	}

	return &receipt, nil
}

// parseTransactionReceipt parses a transaction receipt from raw JSON map
func parseTransactionReceipt(raw map[string]interface{}) TransactionReceipt {
	receipt := TransactionReceipt{}

	if txHash, ok := raw["transactionHash"].(string); ok {
		receipt.TransactionHash = txHash
	}
	if txIndex, ok := raw["transactionIndex"].(string); ok {
		receipt.TransactionIndex = txIndex
	}
	if blockNum, ok := raw["blockNumber"].(string); ok {
		receipt.BlockNumber = blockNum
	}
	if blockHash, ok := raw["blockHash"].(string); ok {
		receipt.BlockHash = blockHash
	}
	if from, ok := raw["from"].(string); ok {
		receipt.From = from
	}
	if to, ok := raw["to"].(string); ok {
		receipt.To = to
	}
	if contractAddr, ok := raw["contractAddress"].(string); ok && contractAddr != "" && contractAddr != "0x" {
		receipt.ContractAddress = contractAddr
	}
	if gasUsed, ok := raw["gasUsed"].(string); ok {
		receipt.GasUsed = gasUsed
	}
	if cumGasUsed, ok := raw["cumulativeGasUsed"].(string); ok {
		receipt.CumulativeGasUsed = cumGasUsed
	}
	if status, ok := raw["status"].(string); ok {
		receipt.Status = status
	}
	if logs, ok := raw["logs"].([]interface{}); ok {
		receipt.Logs = logs
	}

	return receipt
}

// ComputeContractAddress computes the contract address from sender address and nonce
// Formula: keccak256(rlp_encode([sender_address, nonce]))[12:]
// This is a simplified version - for accurate results, use the contract address from transaction receipt
func ComputeContractAddress(sender string, nonce uint64) string {
	// Remove 0x prefix if present
	sender = strings.TrimPrefix(sender, "0x")
	
	// Pad sender address to 20 bytes (40 hex chars)
	if len(sender) < 40 {
		sender = strings.Repeat("0", 40-len(sender)) + sender
	}
	sender = sender[len(sender)-40:] // Take last 40 chars
	
	// Convert to bytes
	senderBytes, _ := hex.DecodeString(sender)
	
	// RLP encode: [address (20 bytes), nonce (uint64)]
	// Simplified: concatenate address + nonce bytes
	nonceBytes := make([]byte, 8)
	for i := 0; i < 8; i++ {
		nonceBytes[7-i] = byte(nonce >> (i * 8))
	}
	
	// Combine: address (20 bytes) + nonce (8 bytes)
	data := append(senderBytes, nonceBytes...)
	
	// Keccak256 hash
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(data)
	result := hasher.Sum(nil)
	
	// Take last 20 bytes (40 hex chars)
	addressBytes := result[len(result)-20:]
	return "0x" + hex.EncodeToString(addressBytes)
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
	// Handle to field - can be string or null
	if toRaw, ok := raw["to"]; ok && toRaw != nil {
		if to, ok := toRaw.(string); ok {
			tx.To = to
		}
	}
	// Check if this is a contract deployment (to is empty/null and input is present)
	if tx.To == "" || tx.To == "0x" || tx.To == "0x0000000000000000000000000000000000000000" {
		if input, ok := raw["input"].(string); ok && input != "" && input != "0x" {
			tx.IsContractDeployment = true
			// Contract address will be computed after nonce is parsed
		}
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

	// If this is a contract deployment and we haven't computed the address yet, do it now
	if tx.IsContractDeployment && tx.ContractAddress == "" {
		tx.ContractAddress = ComputeContractAddress(tx.From, tx.Nonce)
	}

	return tx
}

