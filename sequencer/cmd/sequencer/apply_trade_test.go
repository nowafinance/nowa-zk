package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/nowafinance/nowa-zk/sequencer/internal/batcher"
	"github.com/nowafinance/nowa-zk/sequencer/internal/engine"
	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestTree(t *testing.T) *state.LevelDBMerkleTree {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "sequencer_apply_trade_*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	tree, err := state.NewLevelDBMerkleTree(tmpDir, 28)
	require.NoError(t, err)
	t.Cleanup(func() { tree.Close() })
	return tree
}

func balanceOf(t *testing.T, tree *state.LevelDBMerkleTree, pubKeyHex string, tokenID uint32) *big.Int {
	t.Helper()
	acc, err := getOrCreateBalance(tree, pubKeyHex, tokenID)
	require.NoError(t, err)
	return new(big.Int).Set(acc.Balance)
}

func circuitAuthSig(t *testing.T, priv *eddsa.PrivateKey, opType int64, pubX, pubY, baseIdx, quoteIdx *big.Int) string {
	t.Helper()
	h := mimc.NewMiMC()
	for _, v := range []*big.Int{big.NewInt(opType), pubX, pubY, baseIdx, quoteIdx} {
		var e fr.Element
		e.SetBigInt(v)
		b := e.Bytes()
		h.Write(b[:])
	}
	var msgFr fr.Element
	msgFr.SetBytes(h.Sum(nil))
	msg := msgFr.Bytes()
	sig, err := priv.Sign(msg[:], mimc.NewMiMC())
	require.NoError(t, err)
	return "0x" + hex.EncodeToString(sig)
}

func mustSignedOrder(t *testing.T, tree *state.LevelDBMerkleTree, tokenID uint32, amount, price int64, isBuy bool, nonce uint64) (*types.Order, string, *eddsa.PrivateKey) {
	t.Helper()
	privKey, err := eddsa.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pubHex := "0x" + hex.EncodeToString(privKey.PublicKey.Bytes())
	order := &types.Order{
		MakerAddress: pubHex,
		TokenID:      tokenID,
		Amount:       big.NewInt(amount),
		Price:        big.NewInt(price),
		IsBuy:        isBuy,
		Nonce:        nonce,
	}

	msgStr := fmt.Sprintf("%s:%d:%s:%s:%t:%d",
		order.MakerAddress, order.TokenID, order.Amount.String(), order.Price.String(), order.IsBuy, order.Nonce)
	msgHash := sha256.Sum256([]byte(msgStr))
	msgHash[0] = 0
	sig, err := privKey.Sign(msgHash[:], mimc.NewMiMC())
	require.NoError(t, err)
	order.Signature = hex.EncodeToString(sig)

	// Open leaves and circuit-auth sign (base=tokenID book, quote=2 if base=1).
	baseAcc, err := getOrCreateBalance(tree, pubHex, tokenID)
	require.NoError(t, err)
	quoteTok := uint32(2)
	if tokenID == 2 {
		quoteTok = 1
	}
	quoteAcc, err := getOrCreateBalance(tree, pubHex, quoteTok)
	require.NoError(t, err)
	baseIdx := new(big.Int).SetUint64(baseAcc.AccountID*256 + uint64(baseAcc.TokenID))
	quoteIdx := new(big.Int).SetUint64(quoteAcc.AccountID*256 + uint64(quoteAcc.TokenID))
	order.CircuitSignature = circuitAuthSig(t, privKey, 0, baseAcc.PubKeyX, baseAcc.PubKeyY, baseIdx, quoteIdx)

	return order, pubHex, privKey
}

func TestApplyTrade_PartialFillBalances(t *testing.T) {
	tree := newTestTree(t)
	b := batcher.NewBatcher()
	tradeQueue := make(chan types.Trade, 4)
	eng := engine.NewEngine(tradeQueue)

	// Sell 50, buy 30 → match 30, remainder 20 stays on book.
	ask, sellerPub, _ := mustSignedOrder(t, tree, 1, 50, 50, false, 0)
	bid, buyerPub, _ := mustSignedOrder(t, tree, 1, 30, 50, true, 0)

	sellerStart1 := balanceOf(t, tree, sellerPub, 1)
	sellerStart2 := balanceOf(t, tree, sellerPub, 2)
	buyerStart1 := balanceOf(t, tree, buyerPub, 1)
	buyerStart2 := balanceOf(t, tree, buyerPub, 2)

	require.NoError(t, eng.ProcessOrder(ask))
	require.NoError(t, eng.ProcessOrder(bid))

	var trade types.Trade
	select {
	case trade = <-tradeQueue:
	default:
		t.Fatal("expected a matched trade")
	}
	assert.Equal(t, big.NewInt(30), trade.MatchAmount)
	assert.Equal(t, big.NewInt(50), trade.MatchPrice)

	applyTrade(trade, tree, b)
	assert.Equal(t, 1, b.GetCurrentBatchSize(), "the fill should be tracked as a transition in the open batch")

	quote := new(big.Int).Mul(big.NewInt(30), big.NewInt(50))
	assert.Equal(t, new(big.Int).Sub(sellerStart1, big.NewInt(30)).String(), balanceOf(t, tree, sellerPub, 1).String())
	assert.Equal(t, new(big.Int).Add(sellerStart2, quote).String(), balanceOf(t, tree, sellerPub, 2).String())
	assert.Equal(t, new(big.Int).Add(buyerStart1, big.NewInt(30)).String(), balanceOf(t, tree, buyerPub, 1).String())
	assert.Equal(t, new(big.Int).Sub(buyerStart2, quote).String(), balanceOf(t, tree, buyerPub, 2).String())

	bids, asks := eng.SnapshotOrderbook(1)
	assert.Len(t, bids, 0)
	require.Len(t, asks, 1)
	assert.Equal(t, big.NewInt(20), asks[0].Amount, "remainder stays on book")
}

func TestApplyTrade_RejectsMissingCircuitSig(t *testing.T) {
	tree := newTestTree(t)
	b := batcher.NewBatcher()
	before := tree.Root().String()

	trade := types.Trade{
		TradeID: "tr_no_csig",
		MakerOrder: types.Order{
			MakerAddress: "0x" + hex.EncodeToString(make([]byte, 32)),
			TokenID:      1,
			Amount:       big.NewInt(1),
			Price:        big.NewInt(1),
			IsBuy:        false,
			Signature:    "aa",
		},
		TakerOrder: types.Order{
			MakerAddress: "0x" + hex.EncodeToString(bytesFilled(32, 2)),
			TokenID:      1,
			Amount:       big.NewInt(1),
			Price:        big.NewInt(1),
			IsBuy:        true,
			Signature:    "bb",
		},
		MatchPrice:  big.NewInt(1),
		MatchAmount: big.NewInt(1),
	}
	applyTrade(trade, tree, b)
	assert.Equal(t, before, tree.Root().String())
	assert.Equal(t, uint64(0), b.GetBatchCount())
}

func bytesFilled(n int, v byte) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = v
	}
	return b
}
