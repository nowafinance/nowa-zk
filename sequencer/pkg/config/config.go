package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/tannetwork/zk-sequencer/sequencer/internal/sequencer/types"
	"github.com/tannetwork/zk-sequencer/sequencer/pkg/errors"
	"gopkg.in/yaml.v2"
)

// LoadFromEnv loads configuration from environment variables and .env file
func LoadFromEnv() (*types.Config, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	_ = godotenv.Load()

	config := types.DefaultConfig()

	// Load RPC URL (required)
	if rpcURL := os.Getenv("RPC"); rpcURL != "" {
		config.RPCURL = rpcURL
	}

	// Load WebSocket URL (optional)
	if wsURL := os.Getenv("WS"); wsURL != "" {
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

	// Load index starting block (optional)
	if indexFromBlockStr := os.Getenv("INDEX_FROM_BLOCK"); indexFromBlockStr != "" {
		indexFromBlock, err := strconv.ParseUint(indexFromBlockStr, 10, 64)
		if err != nil {
			return nil, errors.ErrConfig("invalid INDEX_FROM_BLOCK", err)
		}
		config.IndexFromBlock = indexFromBlock
	}

	return config, nil
}

// LoadFromYAML loads configuration from a YAML file
func LoadFromYAML(path string) (*types.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := types.DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, errors.ErrConfig("failed to parse YAML config", err)
	}

	return config, nil
}

// LoadWithOverrides loads config from YAML (if exists), Env, and applies command-line overrides
func LoadWithOverrides(configPath, rpcURL, wsURL, stateDBPath string, batchSize, apiPort int) (*types.Config, error) {
	var config *types.Config
	var err error

	// 1. Try to load from YAML if path is provided or default exists
	if configPath != "" {
		config, err = LoadFromYAML(configPath)
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}

	// If no config loaded yet, use defaults
	if config == nil {
		config = types.DefaultConfig()
	}

	// 2. Override with Env vars
	envConfig, _ := LoadFromEnv()
	if envConfig.RPCURL != "" {
		config.RPCURL = envConfig.RPCURL
	}
	if envConfig.WSURL != "" {
		config.WSURL = envConfig.WSURL
	}
	if envConfig.BatchSize > 0 {
		config.BatchSize = envConfig.BatchSize
	}
	if envConfig.APIPort > 0 {
		config.APIPort = envConfig.APIPort
	}
	if envConfig.StateDBPath != "" {
		config.StateDBPath = envConfig.StateDBPath
	}
	if envConfig.IndexFromBlock > 0 {
		config.IndexFromBlock = envConfig.IndexFromBlock
	}

	// 3. Apply command-line overrides
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
