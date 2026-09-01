package da

import (
	"bytes"
	"encoding/json"
	"testing"
)

// realisticStateUpdate mirrors the real shape of sequencer/internal/types.StateUpdate
// — 28 decimal-string Merkle siblings is the bulk of what makes a real transition
// large, so a synthetic fixture needs the same shape to be a meaningful size test.
func realisticStateUpdate(index int) map[string]any {
	var path [28]string
	var bits [28]int
	for i := range path {
		// A real 254-bit field element as a decimal string — same length ballpark as
		// production data, not a placeholder like "0".
		path[i] = "18746329891274091827364798123647981236479812364798123647981236479812364"
		bits[i] = i % 2
	}
	return map[string]any{
		"index": index, "balance": "1000000", "nonce": 3,
		"is_genesis": false, "path": path, "path_bits": bits,
	}
}

func realisticTransition(opType int) map[string]any {
	return map[string]any{
		"op_type": opType, "amount": "500", "quote_amount": "0",
		"maker_pub_key_x": "16989608640411071100604909338571050458666590912680755147388480380336358609474",
		"maker_pub_key_y": "8766042692033453105235660134453132026909223244917461453796124633491415476055",
		"maker_sig":       "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
		"maker_base":      realisticStateUpdate(1),
		"maker_quote":     realisticStateUpdate(2),
		"taker_pub_key_x": "16989608640411071100604909338571050458666590912680755147388480380336358609474",
		"taker_pub_key_y": "8766042692033453105235660134453132026909223244917461453796124633491415476055",
		"taker_sig":       "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
		"taker_base":      realisticStateUpdate(3),
		"taker_quote":     realisticStateUpdate(4),
	}
}

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

// TestEncodeBatchPayload_25OpBatchFitsCompressed is the regression test for the real
// bug found live: raising BatchSize from 1 to 25 (prover/circuits/state_circuit.go)
// grew a real batch's uncompressed JSON payload to ~248KB — past a single blob's
// 126,976-byte capacity, confirmed live on Sepolia ("DA payload too large for one
// blob: 248006 bytes"). This builds a same-shape 25-op fixture and confirms it would
// have failed uncompressed, but succeeds through the real EncodeBatchPayload path
// now that it compresses first.
func TestEncodeBatchPayload_25OpBatchFitsCompressed(t *testing.T) {
	transitions := make([]map[string]any, 25)
	for i := range transitions {
		transitions[i] = realisticTransition(i % 4) // mix of op types, same as live
	}

	uncompressedJSON, err := json.Marshal(transitions)
	if err != nil {
		t.Fatal(err)
	}
	if len(uncompressedJSON) <= MaxPayloadLen {
		t.Fatalf("test fixture isn't actually representative — uncompressed is only %d bytes, want > %d to reproduce the real overflow", len(uncompressedJSON), MaxPayloadLen)
	}

	payload, dataHash, err := EncodeBatchPayload(1, "123", "456", "0", "0", transitions)
	if err != nil {
		t.Fatalf("EncodeBatchPayload failed on a realistic 25-op batch: %v (this is exactly the live failure this test guards against)", err)
	}
	if len(payload) >= len(uncompressedJSON) {
		t.Fatalf("compressed payload (%d bytes) isn't actually smaller than uncompressed (%d bytes)", len(payload), len(uncompressedJSON))
	}
	if (dataHash == [32]byte{}) {
		t.Fatal("expected non-zero data hash")
	}

	// Must still actually fit in one blob, packed and built for real.
	sidecar, _, err := BuildBlobSidecar(payload)
	if err != nil {
		t.Fatalf("BuildBlobSidecar failed on the compressed payload: %v", err)
	}

	// And must decompress back to exactly the original JSON — GzipDecompress is what
	// reconstruct-proof (a separate module) mirrors, so this pins the exact contract.
	unpacked, err := UnpackBlob(&sidecar.Blobs[0])
	if err != nil {
		t.Fatal(err)
	}
	decompressed, err := GzipDecompress(unpacked)
	if err != nil {
		t.Fatalf("GzipDecompress: %v", err)
	}
	var roundTripped struct {
		Transitions json.RawMessage `json:"transitions"`
	}
	if err := json.Unmarshal(decompressed, &roundTripped); err != nil {
		t.Fatalf("decompressed payload isn't valid JSON: %v", err)
	}
	if !bytes.Equal([]byte(roundTripped.Transitions), uncompressedJSON) {
		t.Fatal("round-tripped transitions don't match the original — compression isn't lossless here")
	}
}
