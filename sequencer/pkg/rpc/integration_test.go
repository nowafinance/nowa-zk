//go:build integration
// +build integration

package rpc

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestIntegration_BlockNumber tests BlockNumber against a real RPC endpoint
func TestIntegration_BlockNumber(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := NewClientFromEnv()
	if err != nil {
		t.Skipf("Skipping integration test: failed to create client from env: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	blockNum, err := client.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("BlockNumber() error = %v", err)
	}

	if blockNum == 0 {
		t.Error("BlockNumber() returned 0, expected non-zero block number")
	}

	t.Logf("Latest block number: %d", blockNum)
}

// TestIntegration_ChainID tests ChainID against a real RPC endpoint
func TestIntegration_ChainID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client, err := NewClientFromEnv()
	if err != nil {
		t.Skipf("Skipping integration test: failed to create client from env: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("ChainID() error = %v", err)
	}

	if chainID == 0 {
		t.Error("ChainID() returned 0, expected non-zero chain ID")
	}

	t.Logf("Chain ID: %d", chainID)
}

// TestIntegration_Connection tests basic connection to RPC endpoint
func TestIntegration_Connection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	rpcURL := os.Getenv("RPC")
	if rpcURL == "" {
		t.Skip("Skipping integration test: RPC not set")
	}

	client, err := NewClient(rpcURL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test connection by calling a simple method
	chainID, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("Failed to connect to RPC endpoint: %v", err)
	}

	t.Logf("Successfully connected to RPC endpoint. Chain ID: %d", chainID)
}

// TestIntegration_RetryLogic tests retry logic with real network conditions
func TestIntegration_RetryLogic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	rpcURL := os.Getenv("RPC")
	if rpcURL == "" {
		t.Skip("Skipping integration test: RPC not set")
	}

	// Create client with aggressive retry settings
	client, err := NewClient(rpcURL, WithMaxRetries(2), WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should succeed even if there are transient network issues
	blockNum, err := client.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("BlockNumber() with retries error = %v", err)
	}

	t.Logf("BlockNumber() succeeded with retries. Block: %d", blockNum)
}
