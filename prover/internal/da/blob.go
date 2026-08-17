// Package da builds EIP-4844 blob sidecars for L1 data availability.
package da

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

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

// EncodeBatchPayload marshals a DA payload and returns raw bytes + keccak256 hash.
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
	payload, err = json.Marshal(p)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("marshal DA payload: %w", err)
	}
	// 4-byte length prefix is included inside the packed blob data budget.
	if len(payload)+4 > MaxPayloadLen {
		return nil, common.Hash{}, fmt.Errorf("DA payload too large for one blob: %d bytes (max %d)", len(payload)+4, MaxPayloadLen)
	}
	return payload, crypto.Keccak256Hash(payload), nil
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
