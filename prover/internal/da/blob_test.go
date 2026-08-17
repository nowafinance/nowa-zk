package da

import (
	"bytes"
	"testing"
)

func TestEncodeAndBuildBlob(t *testing.T) {
	transitions := []map[string]any{
		{"op_type": 0, "amount": "100", "quote_amount": "200"},
	}
	payload, dataHash, err := EncodeBatchPayload(1, "0x01", "0x02", "0", "0", transitions)
	if err != nil {
		t.Fatal(err)
	}
	if (dataHash == [32]byte{}) {
		t.Fatal("expected non-zero data hash")
	}

	sidecar, blobHash, err := BuildBlobSidecar(payload)
	if err != nil {
		t.Fatal(err)
	}
	if sidecar == nil || len(sidecar.Blobs) != 1 {
		t.Fatalf("expected 1 blob, got %#v", sidecar)
	}
	if blobHash[0] != 0x01 {
		t.Fatalf("versioned hash should start with 0x01, got %x", blobHash[0])
	}

	got, err := UnpackBlob(&sidecar.Blobs[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("round-trip mismatch: got %d bytes want %d", len(got), len(payload))
	}
}
