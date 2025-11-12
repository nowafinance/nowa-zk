package types

import "time"

// Config holds sequencer configuration
type Config struct {
	RPCURL          string
	WSURL           string
	BatchSize       int
	BatchInterval   time.Duration
	APIPort         int
	StateDBPath     string
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		BatchSize:     100,              // 100 transactions per batch
		BatchInterval: 5 * time.Second,  // Check/create batch every 5 seconds (reduced from 10s)
		APIPort:       8080,              // REST API port
		StateDBPath:   "./data",         // Local storage path
	}
}

