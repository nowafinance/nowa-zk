package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

// Client is a JSON-RPC client for Nowa-ZK blockchain
type Client struct {
	config           *Config
	httpClient       *http.Client
	url              string
	requestIDCounter atomic.Int64
}

// NewClient creates a new RPC client with the given configuration
func NewClient(rpcURL string, opts ...Option) (*Client, error) {
	config := DefaultConfig()
	config.RPCURL = rpcURL

	// Apply options
	for _, opt := range opts {
		opt(config)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
		url:        rpcURL,
	}, nil
}

// Option is a function that modifies the client configuration
type Option func(*Config)

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(maxRetries int) Option {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

// Close closes the client and releases resources
func (c *Client) Close() error {
	// HTTP client doesn't need explicit closing, but we can clean up here if needed
	return nil
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError represents an RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// call performs a JSON-RPC call with retry logic
func (c *Client) call(ctx context.Context, method string, params []interface{}) (*RPCResponse, error) {
	requestID := int(c.requestIDCounter.Add(1)) // Thread-safe request ID generation
	req := RPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      requestID,
	}

	requestBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay
			backoff := c.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				// Continue with retry
			}
		}

		resp, err := c.doRequest(ctx, requestBody)
		if err != nil {
			lastErr = err
			continue // Retry on error
		}

		if resp.Error != nil {
			return nil, ErrRPCError{
				Code:    resp.Error.Code,
				Message: resp.Error.Message,
				Data:    resp.Error.Data,
			}
		}

		return resp, nil
	}

	return nil, ErrMaxRetriesExceeded{
		Attempts: c.config.MaxRetries + 1,
		LastErr:  lastErr,
	}
}

// doRequest performs a single HTTP request
func (c *Client) doRequest(ctx context.Context, body []byte) (*RPCResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.url, bytes.NewBuffer(body))
	if err != nil {
		return nil, ErrConnectionFailed{URL: c.url, Cause: err}
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrConnectionFailed{URL: c.url, Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(responseBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &rpcResp, nil
}

// calculateBackoff calculates the backoff delay for retry attempts
func (c *Client) calculateBackoff(attempt int) time.Duration {
	backoff := c.config.RetryBackoff
	for i := 0; i < attempt-1; i++ {
		backoff *= 2
		if backoff > c.config.MaxRetryBackoff {
			backoff = c.config.MaxRetryBackoff
			break
		}
	}
	return backoff
}

// BlockNumber returns the latest block number
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	resp, err := c.call(ctx, "eth_blockNumber", nil)
	if err != nil {
		return 0, err
	}

	var hexStr string
	if err := json.Unmarshal(resp.Result, &hexStr); err != nil {
		return 0, fmt.Errorf("failed to unmarshal block number: %w", err)
	}

	// Parse hex string to uint64
	var blockNum uint64
	if _, err := fmt.Sscanf(hexStr, "0x%x", &blockNum); err != nil {
		return 0, fmt.Errorf("failed to parse block number: %w", err)
	}

	return blockNum, nil
}

// ChainID returns the chain ID of the connected network
func (c *Client) ChainID(ctx context.Context) (uint64, error) {
	resp, err := c.call(ctx, "eth_chainId", nil)
	if err != nil {
		return 0, err
	}

	var hexStr string
	if err := json.Unmarshal(resp.Result, &hexStr); err != nil {
		return 0, fmt.Errorf("failed to unmarshal chain ID: %w", err)
	}

	// Parse hex string to uint64
	var chainID uint64
	if _, err := fmt.Sscanf(hexStr, "0x%x", &chainID); err != nil {
		return 0, fmt.Errorf("failed to parse chain ID: %w", err)
	}

	return chainID, nil
}
