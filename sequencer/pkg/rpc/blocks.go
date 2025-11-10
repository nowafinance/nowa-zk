package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
)

// BlockHeader represents a block header
type BlockHeader struct {
	Number     *big.Int `json:"number"`
	Hash       string   `json:"hash"`
	ParentHash string   `json:"parentHash"`
	Timestamp  *big.Int `json:"timestamp"`
	StateRoot  string   `json:"stateRoot"`
}

// Block represents a full block with transactions
type Block struct {
	Number       *big.Int       `json:"number"`
	Hash         string         `json:"hash"`
	ParentHash   string         `json:"parentHash"`
	Timestamp    *big.Int       `json:"timestamp"`
	StateRoot    string         `json:"stateRoot"`
	Transactions []*Transaction `json:"transactions"`
}

// SubscribeNewHeads subscribes to new block headers via WebSocket
// Returns a channel that receives new block headers
func (c *Client) SubscribeNewHeads(ctx context.Context) (<-chan *BlockHeader, error) {
	// For now, we'll use polling with HTTP client
	// WebSocket subscription will be implemented when WebSocket client is integrated
	return nil, fmt.Errorf("WebSocket subscription not yet integrated with HTTP client")
}

// SubscribeNewHeadsWS subscribes to new block headers via WebSocket client
func (ws *WebSocketClient) SubscribeNewHeads(ctx context.Context) (<-chan *BlockHeader, error) {
	ch, err := ws.Subscribe("eth_subscribe", []interface{}{"newHeads"})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	headerChan := make(chan *BlockHeader, 100)

	go func() {
		defer close(headerChan)

		for {
			select {
			case <-ctx.Done():
				return
			case resp, ok := <-ch:
				if !ok {
					return
				}

				// Handle subscription notification format
				// eth_subscription notifications have structure:
				// {"jsonrpc":"2.0","method":"eth_subscription","params":{"subscription":"0x1","result":{...}}}
				// The resp.Result contains the entire notification as JSON
				var notification map[string]interface{}
				if err := json.Unmarshal(resp.Result, &notification); err == nil {
					if method, ok := notification["method"].(string); ok && method == "eth_subscription" {
						if params, ok := notification["params"].(map[string]interface{}); ok {
							if result, ok := params["result"].(map[string]interface{}); ok {
								header := parseBlockHeader(result)
								select {
								case headerChan <- &header:
								case <-ctx.Done():
									return
								}
								continue
							}
						}
					}
				}

				// Try direct parsing (if result is the block header directly)
				var raw map[string]interface{}
				if err := json.Unmarshal(resp.Result, &raw); err == nil {
					// Check if it's a block header (has number and hash)
					if _, hasNumber := raw["number"]; hasNumber {
						if _, hasHash := raw["hash"]; hasHash {
							header := parseBlockHeader(raw)
							select {
							case headerChan <- &header:
							case <-ctx.Done():
								return
							}
						}
					}
				}
			}
		}
	}()

	return headerChan, nil
}

// parseBlockHeader parses a block header from raw JSON map
func parseBlockHeader(raw map[string]interface{}) BlockHeader {
	header := BlockHeader{}

	if num, ok := raw["number"].(string); ok {
		header.Number = parseHexBigInt(num)
	}
	if hash, ok := raw["hash"].(string); ok {
		header.Hash = hash
	}
	if parentHash, ok := raw["parentHash"].(string); ok {
		header.ParentHash = parentHash
	}
	if timestamp, ok := raw["timestamp"].(string); ok {
		header.Timestamp = parseHexBigInt(timestamp)
	}
	if stateRoot, ok := raw["stateRoot"].(string); ok {
		header.StateRoot = stateRoot
	}

	return header
}

// parseHexBigInt parses a hex string to big.Int
func parseHexBigInt(hexStr string) *big.Int {
	if hexStr == "" || hexStr == "0x" {
		return big.NewInt(0)
	}

	// Remove 0x prefix if present
	if len(hexStr) > 2 && hexStr[:2] == "0x" {
		hexStr = hexStr[2:]
	}

	bigInt := new(big.Int)
	bigInt.SetString(hexStr, 16)
	return bigInt
}

// GetBlockByNumber fetches a block by block number
func (c *Client) GetBlockByNumber(ctx context.Context, blockNumber uint64, fullTx bool) (*Block, error) {
	blockNumHex := fmt.Sprintf("0x%x", blockNumber)
	params := []interface{}{blockNumHex, fullTx}

	resp, err := c.call(ctx, "eth_getBlockByNumber", params)
	if err != nil {
		return nil, err
	}

	var block Block
	if err := json.Unmarshal(resp.Result, &block); err != nil {
		// Try parsing as raw map first
		var raw map[string]interface{}
		if err := json.Unmarshal(resp.Result, &raw); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block: %w", err)
		}
		block = parseBlock(raw, fullTx)
	}

	return &block, nil
}

// GetBlockByHash fetches a block by block hash
func (c *Client) GetBlockByHash(ctx context.Context, blockHash string, fullTx bool) (*Block, error) {
	params := []interface{}{blockHash, fullTx}

	resp, err := c.call(ctx, "eth_getBlockByHash", params)
	if err != nil {
		return nil, err
	}

	var block Block
	if err := json.Unmarshal(resp.Result, &block); err != nil {
		// Try parsing as raw map first
		var raw map[string]interface{}
		if err := json.Unmarshal(resp.Result, &raw); err != nil {
			return nil, fmt.Errorf("failed to unmarshal block: %w", err)
		}
		block = parseBlock(raw, fullTx)
	}

	return &block, nil
}

// parseBlock parses a block from raw JSON map
func parseBlock(raw map[string]interface{}, fullTx bool) Block {
	block := Block{}

	if num, ok := raw["number"].(string); ok {
		block.Number = parseHexBigInt(num)
	}
	if hash, ok := raw["hash"].(string); ok {
		block.Hash = hash
	}
	if parentHash, ok := raw["parentHash"].(string); ok {
		block.ParentHash = parentHash
	}
	if timestamp, ok := raw["timestamp"].(string); ok {
		block.Timestamp = parseHexBigInt(timestamp)
	}
	if stateRoot, ok := raw["stateRoot"].(string); ok {
		block.StateRoot = stateRoot
	}

	if fullTx {
		if txs, ok := raw["transactions"].([]interface{}); ok {
			block.Transactions = make([]*Transaction, 0, len(txs))
			for _, txRaw := range txs {
				if txMap, ok := txRaw.(map[string]interface{}); ok {
					tx := parseTransaction(txMap)
					// parseTransaction already handles contract deployment detection and computes address
					// So we just append the transaction
					block.Transactions = append(block.Transactions, &tx)
				}
			}
		}
	}

	return block
}

