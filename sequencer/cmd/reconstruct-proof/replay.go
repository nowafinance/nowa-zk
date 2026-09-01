package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/big"

	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

const (
	blobLimbs    = 4096
	bytesPerLimb = 31
)

// unpackBlob mirrors prover/internal/da/blob.go's UnpackBlob exactly. Duplicated,
// not imported: sequencer and prover are separate Go modules with no dependency
// path between them, and this is ~15 lines — not worth restructuring module
// boundaries for. Keep in sync with the prover-side version if the packing format
// ever changes.
func unpackBlob(blob [blobLimbs * 32]byte) ([]byte, error) {
	raw := make([]byte, 0, blobLimbs*bytesPerLimb)
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

// daPayload mirrors prover/internal/da/blob.go's DAPayload exactly (see unpackBlob's
// comment for why this is duplicated rather than imported).
type daPayload struct {
	Version        uint8                   `json:"version"`
	BatchID        uint64                  `json:"batch_id"`
	OldRoot        string                  `json:"old_root"`
	NewRoot        string                  `json:"new_root"`
	WithdrawalHash string                  `json:"withdrawal_hash"`
	DepositHash    string                  `json:"deposit_hash"`
	Transitions    []types.StateTransition `json:"transitions"`
}

const (
	opTrade      = 0
	opTransfer   = 1
	opWithdrawal = 2
	opDeposit    = 3
)

// replayTransition applies one StateTransition's balance arithmetic to tree — a
// native-Go mirror of prover/circuits/state_circuit.go's processOperation (just the
// balance-update logic, not the ZK constraint verification — this data already
// passed that check when the batch was originally proven, replaying it again here
// would just be redundant work).
//
// Active legs per OpType, matching processOperation exactly:
//
//	makerBaseActive  = OpType != Deposit     (Trade, Transfer, Withdrawal)
//	makerQuoteActive = OpType == Trade
//	takerBaseActive  = OpType != Withdrawal  (Trade, Transfer, Deposit)
//	takerQuoteActive = OpType == Trade
//
// makerBase's nonce bumps by 1 unless it's a Trade (partial-fill reuse, see
// state_circuit.go's comment on why trades don't bump nonce); every other active
// leg's nonce is left unchanged — matches the circuit exactly, not an approximation.
func replayTransition(tree *state.LevelDBMerkleTree, st types.StateTransition) error {
	amount, ok := new(big.Int).SetString(st.Amount, 10)
	if !ok {
		return fmt.Errorf("invalid amount %q", st.Amount)
	}
	quoteAmount, ok := new(big.Int).SetString(st.QuoteAmount, 10)
	if !ok {
		return fmt.Errorf("invalid quote_amount %q", st.QuoteAmount)
	}

	makerBaseActive := st.OpType != opDeposit
	makerQuoteActive := st.OpType == opTrade
	takerBaseActive := st.OpType != opWithdrawal
	takerQuoteActive := st.OpType == opTrade

	if makerBaseActive {
		nonceDelta := uint64(1)
		if st.OpType == opTrade {
			nonceDelta = 0
		}
		if err := applyLeg(tree, st.MakerBase, st.MakerPubKeyX, st.MakerPubKeyY, new(big.Int).Neg(amount), nonceDelta); err != nil {
			return fmt.Errorf("maker base (index %d): %w", st.MakerBase.Index, err)
		}
	}
	if makerQuoteActive {
		if err := applyLeg(tree, st.MakerQuote, st.MakerPubKeyX, st.MakerPubKeyY, quoteAmount, 0); err != nil {
			return fmt.Errorf("maker quote (index %d): %w", st.MakerQuote.Index, err)
		}
	}
	if takerBaseActive {
		if err := applyLeg(tree, st.TakerBase, st.TakerPubKeyX, st.TakerPubKeyY, amount, 0); err != nil {
			return fmt.Errorf("taker base (index %d): %w", st.TakerBase.Index, err)
		}
	}
	if takerQuoteActive {
		if err := applyLeg(tree, st.TakerQuote, st.TakerPubKeyX, st.TakerPubKeyY, new(big.Int).Neg(quoteAmount), 0); err != nil {
			return fmt.Errorf("taker quote (index %d): %w", st.TakerQuote.Index, err)
		}
	}
	return nil
}

// applyLeg writes su.Index's leaf as its OLD balance/nonce (as recorded in the
// transition — the same pre-image the circuit verified inclusion of) plus
// delta/nonceDelta applied. AccountID/TokenID are derived directly from the index
// (accountID = index/256, tokenID = index%256) — no pubkey-hex lookup needed for
// replay itself, only later when a caller wants to query a proof by their own pubkey.
func applyLeg(tree *state.LevelDBMerkleTree, su types.StateUpdate, pubXStr, pubYStr string, delta *big.Int, nonceDelta uint64) error {
	oldBalance, ok := new(big.Int).SetString(su.Balance, 10)
	if !ok {
		return fmt.Errorf("invalid balance %q", su.Balance)
	}
	pubX, ok := new(big.Int).SetString(pubXStr, 10)
	if !ok {
		return fmt.Errorf("invalid pubX %q", pubXStr)
	}
	pubY, ok := new(big.Int).SetString(pubYStr, 10)
	if !ok {
		return fmt.Errorf("invalid pubY %q", pubYStr)
	}

	newBalance := new(big.Int).Add(oldBalance, delta)
	if newBalance.Sign() < 0 {
		return fmt.Errorf("replay produced a negative balance — payload/replay logic mismatch")
	}

	return tree.SetBalance(&types.BalanceState{
		AccountID: su.Index / 256,
		TokenID:   uint32(su.Index % 256),
		PubKeyX:   pubX,
		PubKeyY:   pubY,
		Balance:   newBalance,
		Nonce:     su.Nonce + nonceDelta,
	})
}

// decodeDAPayload unmarshals the unpacked blob payload JSON.
// gzipDecompress mirrors prover/internal/da/blob.go's GzipDecompress exactly —
// duplicated, not imported, for the same cross-module-boundary reason unpackBlob is
// (see its doc comment above). Keep in sync if the compression format ever changes.
func gzipDecompress(compressed []byte) ([]byte, error) {
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

// decodeDAPayload reverses EncodeBatchPayload: gzip-decompress (see
// prover/internal/da/blob.go's EncodeBatchPayload doc comment for why every payload
// is compressed — raising BatchSize past 1 outgrew a single blob's raw capacity),
// then parse the resulting JSON.
func decodeDAPayload(raw []byte) (*daPayload, error) {
	decompressed, err := gzipDecompress(raw)
	if err != nil {
		return nil, fmt.Errorf("decompress DA payload: %w", err)
	}
	var p daPayload
	if err := json.Unmarshal(decompressed, &p); err != nil {
		return nil, fmt.Errorf("decode DA payload: %w", err)
	}
	return &p, nil
}
