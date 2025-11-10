package rpc

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// LoadConfigFromEnv loads configuration from environment variables
// It will also try to load from .env file if it exists
func LoadConfigFromEnv() (*Config, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	_ = godotenv.Load()

	rpcURL := os.Getenv("TAN_ZK_RPC_URL")
	if rpcURL == "" {
		return nil, fmt.Errorf("TAN_ZK_RPC_URL environment variable is required")
	}

	config := DefaultConfig()
	config.RPCURL = rpcURL

	// Load optional timeout
	if timeoutStr := os.Getenv("RPC_TIMEOUT"); timeoutStr != "" {
		timeout, err := strconv.Atoi(timeoutStr)
		if err != nil {
			return nil, fmt.Errorf("invalid RPC_TIMEOUT: %w", err)
		}
		config.Timeout = time.Duration(timeout) * time.Second
	}

	// Load optional max retries
	if retriesStr := os.Getenv("RPC_MAX_RETRIES"); retriesStr != "" {
		retries, err := strconv.Atoi(retriesStr)
		if err != nil {
			return nil, fmt.Errorf("invalid RPC_MAX_RETRIES: %w", err)
		}
		config.MaxRetries = retries
	}

	// Load optional retry backoff
	if backoffStr := os.Getenv("RPC_RETRY_BACKOFF"); backoffStr != "" {
		backoff, err := strconv.Atoi(backoffStr)
		if err != nil {
			return nil, fmt.Errorf("invalid RPC_RETRY_BACKOFF: %w", err)
		}
		config.RetryBackoff = time.Duration(backoff) * time.Millisecond
	}

	// Load optional max retry backoff
	if maxBackoffStr := os.Getenv("RPC_MAX_RETRY_BACKOFF"); maxBackoffStr != "" {
		maxBackoff, err := strconv.Atoi(maxBackoffStr)
		if err != nil {
			return nil, fmt.Errorf("invalid RPC_MAX_RETRY_BACKOFF: %w", err)
		}
		config.MaxRetryBackoff = time.Duration(maxBackoff) * time.Second
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// NewClientFromEnv creates a new RPC client from environment variables
func NewClientFromEnv() (*Client, error) {
	config, err := LoadConfigFromEnv()
	if err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
		url:        config.RPCURL,
	}, nil
}

