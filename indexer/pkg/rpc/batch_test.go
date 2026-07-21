package rpc

import (
	"context"
	"testing"
)

func TestSubmitBatchProof_Stub(t *testing.T) {
	client, err := NewClient("http://example.com")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	proof := &BatchProof{
		Proof:        []byte{0x01, 0x02, 0x03},
		PublicInputs: []string{"0x1234", "0x5678"},
		BatchHash:    "0xabcd",
		StateRoot:    "0xef01",
	}

	_, err = client.SubmitBatchProof(context.Background(), proof)
	if err == nil {
		t.Error("SubmitBatchProof() expected error for stub implementation, got nil")
	}
	if err.Error() != "SubmitBatchProof is not yet implemented - contract interaction pending" {
		t.Errorf("SubmitBatchProof() error = %v, want 'not yet implemented'", err)
	}
}

func TestSubmitBatchProof_Validation(t *testing.T) {
	client, err := NewClient("http://example.com")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	tests := []struct {
		name    string
		proof   *BatchProof
		wantErr string
	}{
		{
			name:    "nil proof",
			proof:   nil,
			wantErr: "proof cannot be nil",
		},
		{
			name: "empty batchHash",
			proof: &BatchProof{
				Proof:     []byte{0x01},
				StateRoot: "0x1234",
			},
			wantErr: "batchHash cannot be empty",
		},
		{
			name: "empty stateRoot",
			proof: &BatchProof{
				Proof:     []byte{0x01},
				BatchHash: "0xabcd",
			},
			wantErr: "stateRoot cannot be empty",
		},
		{
			name: "empty proof data",
			proof: &BatchProof{
				BatchHash: "0xabcd",
				StateRoot: "0x1234",
			},
			wantErr: "proof data cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.SubmitBatchProof(context.Background(), tt.proof)
			if err == nil {
				t.Errorf("SubmitBatchProof() expected error '%s', got nil", tt.wantErr)
				return
			}
			if err.Error() != tt.wantErr {
				t.Errorf("SubmitBatchProof() error = %v, want %s", err, tt.wantErr)
			}
		})
	}
}

func TestGetBatchStatus_Stub(t *testing.T) {
	client, err := NewClient("http://example.com")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	_, err = client.GetBatchStatus(context.Background(), 1)
	if err == nil {
		t.Error("GetBatchStatus() expected error for stub implementation, got nil")
	}
}

func TestGetBatchByHash_Stub(t *testing.T) {
	client, err := NewClient("http://example.com")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	_, err = client.GetBatchByHash(context.Background(), "0xabcd")
	if err == nil {
		t.Error("GetBatchByHash() expected error for stub implementation, got nil")
	}
}
