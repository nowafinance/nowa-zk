package rpc_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nowafinance/nowa-zk/sequencer/pkg/rpc"
)

func ExampleNewClientFromEnv() {
	// Load client from environment variables (.env file)
	client, err := rpc.NewClientFromEnv()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Get chain ID
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}
	fmt.Printf("Chain ID: %d\n", chainID)

	// Get latest block number
	blockNum, err := client.BlockNumber(ctx)
	if err != nil {
		log.Fatalf("Failed to get block number: %v", err)
	}
	fmt.Printf("Latest block: %d\n", blockNum)
}

func ExampleNewClient() {
	// Create client with explicit URL
	client, err := rpc.NewClient("http://localhost:8545")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	blockNum, err := client.BlockNumber(ctx)
	if err != nil {
		log.Fatalf("Failed to get block number: %v", err)
	}
	fmt.Printf("Block number: %d\n", blockNum)
}

func ExampleNewClient_withOptions() {
	// Create client with custom configuration
	client, err := rpc.NewClient(
		"http://localhost:8545",
		rpc.WithTimeout(60*time.Second),
		rpc.WithMaxRetries(5),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	chainID, err := client.ChainID(ctx)
	if err != nil {
		log.Fatalf("Failed to get chain ID: %v", err)
	}
	fmt.Printf("Chain ID: %d\n", chainID)
}

func ExampleLoadConfigFromEnv() {
	// Load configuration from environment
	config, err := rpc.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Use the config to create client
	client, err := rpc.NewClient(config.RPCURL)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Printf("Connected to: %s\n", config.RPCURL)
}

func init() {
	// Set a test RPC URL if not already set (for example tests)
	if os.Getenv("RPC") == "" {
		os.Setenv("RPC", "http://localhost:8545")
	}
}
