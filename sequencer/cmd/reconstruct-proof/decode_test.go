package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"testing"

	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

// TestDecodeDAPayload_ReversesGzipCompression is the reconstruct-proof-side half of
// the regression test for the DA blob-size bug: raising BatchSize to 25 outgrew a
// single blob uncompressed, so every payload the Prover writes is now gzip-compressed
// (see prover/internal/da/blob.go's EncodeBatchPayload). This confirms
// decodeDAPayload correctly reverses that — decompress then parse — using a fixture
// compressed the same way (gzip.NewWriter), independent of the Prover module.
func TestDecodeDAPayload_ReversesGzipCompression(t *testing.T) {
	original := daPayload{
		Version: 1, BatchID: 7, OldRoot: "111", NewRoot: "222",
		WithdrawalHash: "0", DepositHash: "0",
		Transitions: []types.StateTransition{{OpType: 3, Amount: "500"}},
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	got, err := decodeDAPayload(buf.Bytes())
	if err != nil {
		t.Fatalf("decodeDAPayload: %v", err)
	}
	if got.BatchID != 7 || got.OldRoot != "111" || got.NewRoot != "222" {
		t.Fatalf("decoded payload doesn't match original: %+v", got)
	}
	if len(got.Transitions) != 1 || got.Transitions[0].Amount != "500" {
		t.Fatalf("decoded transitions don't match original: %+v", got.Transitions)
	}
}

func TestDecodeDAPayload_RejectsUncompressedInput(t *testing.T) {
	// Plain (non-gzip) JSON must be rejected, not silently misparsed — every payload
	// from the real Prover is compressed now, so anything else is unexpected input.
	raw, _ := json.Marshal(daPayload{BatchID: 1})
	if _, err := decodeDAPayload(raw); err == nil {
		t.Fatal("expected an error decoding uncompressed input, got nil")
	}
}

// TestReadCheckpoint_RejectsCorruptData is the regression test for a Copilot review
// finding: a truncated or otherwise malformed persisted checkpoint value used to
// panic binary.BigEndian.Uint64 (index-out-of-range) instead of failing gracefully.
func TestReadCheckpoint_RejectsCorruptData(t *testing.T) {
	tree := newTestTree(t)

	// No checkpoint written yet — 0, no error.
	got, err := readCheckpoint(tree)
	if err != nil || got != 0 {
		t.Fatalf("fresh tree: got (%d, %v), want (0, nil)", got, err)
	}

	// A truncated value (not 8 bytes) must error, not panic.
	if err := tree.SetMeta(lastReplayedBatchKey, []byte{0x01, 0x02, 0x03}); err != nil {
		t.Fatal(err)
	}
	if _, err := readCheckpoint(tree); err == nil {
		t.Fatal("expected an error for a truncated (3-byte) checkpoint value, got nil")
	}
}

// TestGzipDecompress_RejectsOversizedPayload is the regression test for the
// decompression-bomb Copilot finding: a small compressed input that expands past
// maxDecompressedPayloadLen must error, not silently allocate unbounded memory.
func TestGzipDecompress_RejectsOversizedPayload(t *testing.T) {
	// Highly compressible: all zeros, expands well past the cap from a small input.
	huge := make([]byte, maxDecompressedPayloadLen+1024)

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(huge); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := gzipDecompress(buf.Bytes()); err == nil {
		t.Fatal("expected an error decompressing a payload over the size cap, got nil")
	}
}
