package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		rpcURL  string
		opts    []Option
		wantErr bool
	}{
		{
			name:    "valid URL",
			rpcURL:  "http://localhost:8545",
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "valid URL with timeout",
			rpcURL:  "http://localhost:8545",
			opts:    []Option{WithTimeout(60 * time.Second)},
			wantErr: false,
		},
		{
			name:    "empty URL",
			rpcURL:  "",
			opts:    nil,
			wantErr: true,
		},
		{
			name:    "invalid timeout",
			rpcURL:  "http://localhost:8545",
			opts:    []Option{WithTimeout(-1 * time.Second)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.rpcURL, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client without error")
			}
			if client != nil {
				client.Close()
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				RPCURL:          "http://localhost:8545",
				Timeout:         30 * time.Second,
				MaxRetries:      3,
				RetryBackoff:    100 * time.Millisecond,
				MaxRetryBackoff: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty RPCURL",
			config: &Config{
				RPCURL: "",
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: &Config{
				RPCURL:  "http://localhost:8545",
				Timeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative max retries",
			config: &Config{
				RPCURL:     "http://localhost:8545",
				MaxRetries: -1,
			},
			wantErr: true,
		},
		{
			name: "max backoff less than backoff",
			config: &Config{
				RPCURL:          "http://localhost:8545",
				RetryBackoff:    10 * time.Second,
				MaxRetryBackoff: 1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_BlockNumber(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`"0x1234"`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	blockNum, err := client.BlockNumber(context.Background())
	if err != nil {
		t.Fatalf("BlockNumber() error = %v", err)
	}

	expected := uint64(0x1234)
	if blockNum != expected {
		t.Errorf("BlockNumber() = %d, want %d", blockNum, expected)
	}
}

func TestClient_ChainID(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`"0x1"`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		t.Fatalf("ChainID() error = %v", err)
	}

	expected := uint64(1)
	if chainID != expected {
		t.Errorf("ChainID() = %d, want %d", chainID, expected)
	}
}

func TestClient_RetryLogic(t *testing.T) {
	attempts := 0
	// Create a mock server that fails twice then succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		response := RPCResponse{
			JSONRPC: "2.0",
			ID:      1,
			Result:  json.RawMessage(`"0x1234"`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, WithMaxRetries(3))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	blockNum, err := client.BlockNumber(context.Background())
	if err != nil {
		t.Fatalf("BlockNumber() error = %v", err)
	}

	expected := uint64(0x1234)
	if blockNum != expected {
		t.Errorf("BlockNumber() = %d, want %d", blockNum, expected)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestClient_MaxRetriesExceeded(t *testing.T) {
	// Create a mock server that always fails
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, WithMaxRetries(2))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	_, err = client.BlockNumber(context.Background())
	if err == nil {
		t.Error("BlockNumber() expected error, got nil")
	}

	var maxRetriesErr ErrMaxRetriesExceeded
	if !isErrorType(err, &maxRetriesErr) {
		t.Errorf("BlockNumber() error = %v, want ErrMaxRetriesExceeded", err)
	}
}

// Helper function to check error type
func isErrorType(err error, target interface{}) bool {
	// Simple type assertion check
	switch target.(type) {
	case *ErrMaxRetriesExceeded:
		_, ok := err.(ErrMaxRetriesExceeded)
		return ok
	default:
		return false
	}
}

