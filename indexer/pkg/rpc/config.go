package rpc

import (
	"time"
)

// Config holds configuration for the RPC client
type Config struct {
	// RPCURL is the HTTP/HTTPS endpoint URL for JSON-RPC
	RPCURL string

	// Timeout is the maximum time to wait for a response
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// RetryBackoff is the initial backoff delay for retries
	RetryBackoff time.Duration

	// MaxRetryBackoff is the maximum backoff delay
	MaxRetryBackoff time.Duration
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		RetryBackoff:    100 * time.Millisecond,
		MaxRetryBackoff: 10 * time.Second,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.RPCURL == "" {
		return ErrInvalidConfig{Field: "RPCURL", Reason: "cannot be empty"}
	}
	if c.Timeout <= 0 {
		return ErrInvalidConfig{Field: "Timeout", Reason: "must be positive"}
	}
	if c.MaxRetries < 0 {
		return ErrInvalidConfig{Field: "MaxRetries", Reason: "cannot be negative"}
	}
	if c.RetryBackoff <= 0 {
		return ErrInvalidConfig{Field: "RetryBackoff", Reason: "must be positive"}
	}
	if c.MaxRetryBackoff < c.RetryBackoff {
		return ErrInvalidConfig{Field: "MaxRetryBackoff", Reason: "must be >= RetryBackoff"}
	}
	return nil
}

// WithTimeout returns a new config with the specified timeout
func (c *Config) WithTimeout(timeout time.Duration) *Config {
	cfg := *c
	cfg.Timeout = timeout
	return &cfg
}

// WithMaxRetries returns a new config with the specified max retries
func (c *Config) WithMaxRetries(maxRetries int) *Config {
	cfg := *c
	cfg.MaxRetries = maxRetries
	return &cfg
}

