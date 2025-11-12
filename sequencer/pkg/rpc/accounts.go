package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
)

// AccountState represents the state of an account
type AccountState struct {
	Balance  *big.Int `json:"balance"`
	Nonce    uint64   `json:"nonce"`
	Code     string   `json:"code"`     // Contract code (if contract account)
	CodeHash string   `json:"codeHash"` // Hash of contract code
}

// GetBalanceAtBlock fetches the balance of an account at a specific block number
func (c *Client) GetBalanceAtBlock(ctx context.Context, address string, blockNumber uint64) (*big.Int, error) {
	blockNumBig := big.NewInt(int64(blockNumber))
	return c.GetBalance(ctx, address, blockNumBig)
}

// GetBalance fetches the balance of an account
func (c *Client) GetBalance(ctx context.Context, address string, blockNumber *big.Int) (*big.Int, error) {
	var blockTag string
	if blockNumber == nil {
		blockTag = "latest"
	} else {
		blockTag = fmt.Sprintf("0x%x", blockNumber)
	}

	params := []interface{}{address, blockTag}

	resp, err := c.call(ctx, "eth_getBalance", params)
	if err != nil {
		return nil, err
	}

	var balanceHex string
	if err := json.Unmarshal(resp.Result, &balanceHex); err != nil {
		return nil, fmt.Errorf("failed to unmarshal balance: %w", err)
	}

	return parseHexBigInt(balanceHex), nil
}

// GetTransactionCount fetches the nonce (transaction count) of an account
func (c *Client) GetTransactionCount(ctx context.Context, address string, blockNumber *big.Int) (uint64, error) {
	var blockTag string
	if blockNumber == nil {
		blockTag = "latest"
	} else {
		blockTag = fmt.Sprintf("0x%x", blockNumber)
	}

	params := []interface{}{address, blockTag}

	resp, err := c.call(ctx, "eth_getTransactionCount", params)
	if err != nil {
		return 0, err
	}

	var nonceHex string
	if err := json.Unmarshal(resp.Result, &nonceHex); err != nil {
		return 0, fmt.Errorf("failed to unmarshal nonce: %w", err)
	}

	return parseHexBigInt(nonceHex).Uint64(), nil
}

// GetCode fetches the contract code of an account
func (c *Client) GetCode(ctx context.Context, address string, blockNumber *big.Int) (string, error) {
	var blockTag string
	if blockNumber == nil {
		blockTag = "latest"
	} else {
		blockTag = fmt.Sprintf("0x%x", blockNumber)
	}

	params := []interface{}{address, blockTag}

	resp, err := c.call(ctx, "eth_getCode", params)
	if err != nil {
		return "", err
	}

	var code string
	if err := json.Unmarshal(resp.Result, &code); err != nil {
		return "", fmt.Errorf("failed to unmarshal code: %w", err)
	}

	return code, nil
}

// GetAccountState fetches the complete state of an account
func (c *Client) GetAccountState(ctx context.Context, address string, blockNumber *big.Int) (*AccountState, error) {
	// Fetch balance, nonce, and code in parallel would be ideal, but for simplicity we'll do sequentially
	balance, err := c.GetBalance(ctx, address, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	nonce, err := c.GetTransactionCount(ctx, address, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	code, err := c.GetCode(ctx, address, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get code: %w", err)
	}

	// Calculate code hash (simple hash for now, could use keccak256)
	codeHash := ""
	if code != "" && code != "0x" {
		// In a real implementation, this would be keccak256(code)
		// For now, we'll leave it empty or use a placeholder
		codeHash = "0x" // Placeholder
	}

	return &AccountState{
		Balance:  balance,
		Nonce:    nonce,
		Code:     code,
		CodeHash: codeHash,
	}, nil
}

