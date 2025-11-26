package types

import "time"

// Config holds sequencer configuration
type Config struct {
	RPCURL        string        `yaml:"rpc_url"`
	WSURL         string        `yaml:"ws_url"`
	BatchSize     int           `yaml:"batch_size"`
	BatchInterval time.Duration `yaml:"batch_interval"`
	APIPort       int           `yaml:"api_port"`
	StateDBPath   string        `yaml:"state_db_path"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		BatchSize:     100,                      // 100 transactions per batch
		BatchInterval: 5 * time.Second,          // Check/create batch every 5 seconds (reduced from 10s)
		APIPort:       8080,                     // REST API port
		StateDBPath:   ".tan-zk/sequencer/data", // Local storage path
	}
}
