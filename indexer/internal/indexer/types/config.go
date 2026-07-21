package types

import "time"

// Config holds indexer configuration
type Config struct {
	RPCURL         string        `yaml:"rpc_url"`
	WSURL          string        `yaml:"ws_url"`
	BatchSize      int           `yaml:"batch_size"`
	BatchInterval  time.Duration `yaml:"batch_interval"`
	APIPort        int           `yaml:"api_port"`
	StateDBPath    string        `yaml:"state_db_path"`
	IndexFromBlock uint64        `yaml:"index_from_block"`
	ProverAPI      string        `yaml:"prover_api"` // Prover API URL for cleanup coordination
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		BatchSize:     128,                       // 128 transactions per batch
		BatchInterval: 5 * time.Second,           // Check/create batch every 5 seconds (reduced from 10s)
		APIPort:       8080,                      // REST API port
		StateDBPath:   ".nowa-zk/indexer/data", // Local storage path
		ProverAPI:     "http://localhost:8081",   // Default prover API (can run on different server)
	}
}
