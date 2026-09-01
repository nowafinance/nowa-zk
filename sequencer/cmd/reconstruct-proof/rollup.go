package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Minimal hand-written ABI for just the 4 read-only getters this tool needs — same
// pattern already used in claim-escape and prover/cmd/prover/start.go, since the
// checked-in generated bindings (sequencer/internal/bindings) are known-stale and
// don't have these getters either (see docs/project/release-status.md).
const rollupReaderABI = `[
	{"inputs":[],"name":"batchCount","outputs":[{"internalType":"uint64","name":"","type":"uint64"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"uint64","name":"","type":"uint64"}],"name":"batchBlobHash","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"},
	{"inputs":[{"internalType":"uint64","name":"","type":"uint64"}],"name":"batchDataHash","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"},
	{"inputs":[],"name":"stateRoot","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"}
]`

type rollupReader struct {
	bound *bind.BoundContract
}

func newRollupReader(rpcURL, contractAddr string) (*rollupReader, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial L1 RPC: %w", err)
	}
	parsedABI, err := abi.JSON(strings.NewReader(rollupReaderABI))
	if err != nil {
		return nil, fmt.Errorf("parse ABI: %w", err)
	}
	bound := bind.NewBoundContract(common.HexToAddress(contractAddr), parsedABI, client, client, client)
	return &rollupReader{bound: bound}, nil
}

func (r *rollupReader) batchCount() (uint64, error) {
	var out []interface{}
	if err := r.bound.Call(&bind.CallOpts{Context: context.Background()}, &out, "batchCount"); err != nil {
		return 0, err
	}
	return out[0].(uint64), nil
}

func (r *rollupReader) batchBlobHash(batchID uint64) ([32]byte, error) {
	var out []interface{}
	if err := r.bound.Call(&bind.CallOpts{Context: context.Background()}, &out, "batchBlobHash", batchID); err != nil {
		return [32]byte{}, err
	}
	return out[0].([32]byte), nil
}

func (r *rollupReader) batchDataHash(batchID uint64) ([32]byte, error) {
	var out []interface{}
	if err := r.bound.Call(&bind.CallOpts{Context: context.Background()}, &out, "batchDataHash", batchID); err != nil {
		return [32]byte{}, err
	}
	return out[0].([32]byte), nil
}

func (r *rollupReader) stateRoot() ([32]byte, error) {
	var out []interface{}
	if err := r.bound.Call(&bind.CallOpts{Context: context.Background()}, &out, "stateRoot"); err != nil {
		return [32]byte{}, err
	}
	return out[0].([32]byte), nil
}
