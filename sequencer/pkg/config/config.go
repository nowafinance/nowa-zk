package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/tannetwork/zk-sequencer/sequencer/internal/sequencer/types"
	"github.com/tannetwork/zk-sequencer/sequencer/pkg/errors"
)

// LoadFromEnv loads configuration from environment variables and .env file
func LoadFromEnv() (*types.Config, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	_ = godotenv.Load()

	config := types.DefaultConfig()

	// Load RPC URL (required)
	if rpcURL := os.Getenv("TAN_ZK_RPC_URL"); rpcURL != "" {
		config.RPCURL = rpcURL
	}

	// Load WebSocket URL (optional)
	if wsURL := os.Getenv("TAN_ZK_WS_URL"); wsURL != "" {
		config.WSURL = wsURL
	}

	// Load batch size (optional)
	if batchSizeStr := os.Getenv("BATCH_SIZE"); batchSizeStr != "" {
		batchSize, err := strconv.Atoi(batchSizeStr)
		if err != nil {
			return nil, errors.ErrConfig("invalid BATCH_SIZE", err)
		}
		config.BatchSize = batchSize
	}

	// Load batch interval (optional)
	if batchIntervalStr := os.Getenv("BATCH_INTERVAL"); batchIntervalStr != "" {
		batchInterval, err := strconv.Atoi(batchIntervalStr)
		if err != nil {
			return nil, errors.ErrConfig("invalid BATCH_INTERVAL", err)
		}
		config.BatchInterval = time.Duration(batchInterval) * time.Second
	}

	// Load API port (optional)
	if apiPortStr := os.Getenv("API_PORT"); apiPortStr != "" {
		apiPort, err := strconv.Atoi(apiPortStr)
		if err != nil {
			return nil, errors.ErrConfig("invalid API_PORT", err)
		}
		config.APIPort = apiPort
	}

	// Load state DB path (optional)
	if stateDBPath := os.Getenv("STATE_DB_PATH"); stateDBPath != "" {
		config.StateDBPath = stateDBPath
	}

	return config, nil
}

// LoadWithOverrides loads config from env and applies command-line overrides
func LoadWithOverrides(rpcURL, wsURL, stateDBPath string, batchSize, apiPort int) (*types.Config, error) {
	config, err := LoadFromEnv()
	if err != nil {
		return nil, err
	}

	// Apply command-line overrides
	if rpcURL != "" {
		config.RPCURL = rpcURL
	}
	if wsURL != "" {
		config.WSURL = wsURL
	}
	if stateDBPath != "" {
		config.StateDBPath = stateDBPath
	}
	if batchSize > 0 {
		config.BatchSize = batchSize
	}
	if apiPort > 0 {
		config.APIPort = apiPort
	}

	// Validate required fields
	if config.RPCURL == "" {
		return nil, errors.ErrInvalidInput("TAN_ZK_RPC_URL is required")
	}

	return config, nil
}

