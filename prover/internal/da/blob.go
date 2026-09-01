// Package da builds EIP-4844 blob sidecars for L1 data availability.
package da

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
	"github.com/ethereum/go-ethereum/core/types"
)

// Blob field-element packing: 4096 limbs × 31 usable bytes = 126,976 max payload.
const (
	blobLimbs     = 4096
	bytesPerLimb  = 31
	MaxPayloadLen = blobLimbs * bytesPerLimb
)

// DAPayload is the canonical JSON written into blob index 0.
// Anyone can reconstruct batch transitions from this payload + L1 state roots.
type DAPayload struct {
	Version        uint8           `json:"version"`
	BatchID        uint64          `json:"batch_id"`
	OldRoot        string          `json:"old_root"`
	NewRoot        string          `json:"new_root"`
	WithdrawalHash string          `json:"withdrawal_hash"`
	DepositHash    string          `json:"deposit_hash"`
	Transitions    json.RawMessage `json:"transitions"`
}

// EncodeBatchPayload marshals a DA payload, gzip-compresses it, and returns the
// compressed bytes + keccak256 hash of those same compressed bytes (dataHash is a
// hash of exactly what ends up packed into the blob, per submitBatch's "_dataHash
// ... must match bytes stored in the blob" contract — so integrity verification
// never needs to decompress first to check it).
//
// Compression exists because raising BatchSize from 1 to 25 (see
// prover/circuits/state_circuit.go) grew a real batch's uncompressed JSON payload
// past a single blob's 126,976-byte capacity (a 25-op batch measured at ~248KB) —
// confirmed live. The payload is mostly decimal-string-encoded Merkle path siblings,
// which compress well (repetitive, numeric). This is a pragmatic fix, not a
// structural one: it buys headroom rather than removing the one-blob-per-batch
// ceiling entirely — a batch whose data is unusually incompressible, or a further
// increase in BatchSize, could still exceed it. True multi-blob support (EIP-4844
// allows up to 6 blobs per transaction) would remove the ceiling properly, at the
// cost of a NowaRollup.sol change (it currently tracks one blob hash per batch) —
// not done here, deliberately deferred.
func EncodeBatchPayload(batchID uint64, oldRoot, newRoot, withdrawalHash, depositHash string, transitions any) (payload []byte, dataHash common.Hash, err error) {
	transJSON, err := json.Marshal(transitions)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("marshal transitions: %w", err)
	}
	p := DAPayload{
		Version:        1,
		BatchID:        batchID,
		OldRoot:        oldRoot,
		NewRoot:        newRoot,
		WithdrawalHash: withdrawalHash,
		DepositHash:    depositHash,
		Transitions:    transJSON,
	}
	uncompressed, err := json.Marshal(p)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("marshal DA payload: %w", err)
	}
	payload, err = gzipCompress(uncompressed)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("compress DA payload: %w", err)
	}
	// 4-byte length prefix is included inside the packed blob data budget.
	if len(payload)+4 > MaxPayloadLen {
		return nil, common.Hash{}, fmt.Errorf("DA payload too large for one blob even after compression: %d bytes compressed from %d (max %d)", len(payload)+4, len(uncompressed), MaxPayloadLen)
	}
	return payload, crypto.Keccak256Hash(payload), nil
}

func gzipCompress(raw []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(raw); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GzipDecompress reverses gzipCompress — exported so reconstruct-proof (a separate
// Go module, see sequencer/cmd/reconstruct-proof/replay.go) can mirror it exactly
// rather than risk drifting from this implementation.
func GzipDecompress(compressed []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, fmt.Errorf("gzip reader: %w", err)
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("gzip read: %w", err)
	}
	return out, nil
}

// packBlob encodes arbitrary bytes into a valid EIP-4844 blob.
// Each 32-byte limb stores a leading 0x00 byte (so the value is a canonical BLS scalar)
// followed by up to 31 payload bytes.
// The first 4 payload bytes are a big-endian length of the remaining payload.
func packBlob(payload []byte) (kzg4844.Blob, error) {
	var blob kzg4844.Blob
	if len(payload)+4 > MaxPayloadLen {
		return blob, fmt.Errorf("payload exceeds blob capacity")
	}

	raw := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(raw[:4], uint32(len(payload)))
	copy(raw[4:], payload)

	offset := 0
	for i := 0; i < blobLimbs && offset < len(raw); i++ {
		n := bytesPerLimb
		if remain := len(raw) - offset; remain < n {
			n = remain
		}
		// limb[0] stays 0 → field element is canonical
		copy(blob[i*32+1:i*32+1+n], raw[offset:offset+n])
		offset += n
	}
	return blob, nil
}

// UnpackBlob reverses packBlob and returns the original payload bytes.
func UnpackBlob(blob *kzg4844.Blob) ([]byte, error) {
	raw := make([]byte, 0, MaxPayloadLen)
	for i := 0; i < blobLimbs; i++ {
		raw = append(raw, blob[i*32+1:i*32+32]...)
	}
	if len(raw) < 4 {
		return nil, fmt.Errorf("blob too short")
	}
	length := binary.BigEndian.Uint32(raw[:4])
	if int(length) > len(raw)-4 {
		return nil, fmt.Errorf("invalid length prefix %d", length)
	}
	return raw[4 : 4+length], nil
}

// BuildBlobSidecar packs payload into a single EIP-4844 blob and returns a v1 (cell-proof)
// sidecar + versioned hash. Post-Osaka (EIP-7594 / PeerDAS), nodes reject the old v0
// single-proof sidecar wrapper outright ("unexpected eip-4844 sidecar after osaka"), so we
// build cell proofs directly rather than a whole-blob proof.
func BuildBlobSidecar(payload []byte) (*types.BlobTxSidecar, common.Hash, error) {
	blob, err := packBlob(payload)
	if err != nil {
		return nil, common.Hash{}, err
	}

	commitment, err := kzg4844.BlobToCommitment(&blob)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("blob commitment: %w", err)
	}
	cellProofs, err := kzg4844.ComputeCellProofs(&blob)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("blob cell proofs: %w", err)
	}

	sidecar := types.NewBlobTxSidecar(
		types.BlobSidecarVersion1,
		[]kzg4844.Blob{blob},
		[]kzg4844.Commitment{commitment},
		cellProofs,
	)
	hashes := sidecar.BlobHashes()
	if len(hashes) != 1 {
		return nil, common.Hash{}, fmt.Errorf("expected 1 blob hash, got %d", len(hashes))
	}
	if hashes[0][0] != 0x01 {
		return nil, common.Hash{}, fmt.Errorf("invalid versioned hash prefix: %x", hashes[0][0])
	}
	return sidecar, hashes[0], nil
}
