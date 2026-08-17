package engine

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
	"github.com/stretchr/testify/assert"
)

func generateTestOrder(t *testing.T, tokenID uint32, amount, price int64, isBuy bool) (*types.Order, *eddsa.PrivateKey) {
	privKey, err := eddsa.GenerateKey(rand.Reader)
	assert.NoError(t, err)
	pubKey := privKey.PublicKey

	pubBytes := pubKey.Bytes()

	order := &types.Order{
		MakerAddress: "0xTest",
		TokenID:      tokenID,
		Amount:       big.NewInt(amount),
		Price:        big.NewInt(price),
		IsBuy:        isBuy,
		Nonce:        1,
	}

	msgStr := fmt.Sprintf("%s:%d:%s:%s:%t:%d", hex.EncodeToString(pubBytes), order.TokenID, order.Amount.String(), order.Price.String(), order.IsBuy, order.Nonce)
	msgHash := sha256.Sum256([]byte(msgStr))
	msgHash[0] = 0 // ensure < modulus
	
	hFunc := mimc.NewMiMC()
	sig, err := privKey.Sign(msgHash[:], hFunc)
	assert.NoError(t, err)
	order.Signature = hex.EncodeToString(sig)

	order.MakerAddress = hex.EncodeToString(pubBytes)

	return order, privKey
}

func TestEngine_MatchOrders(t *testing.T) {
	tradeQueue := make(chan types.Trade, 10)
	engine := NewEngine(tradeQueue)

	// Create a sell order (Ask) for 100 @ 50
	ask, _ := generateTestOrder(t, 1, 100, 50, false)
	err := engine.ProcessOrder(ask)
	assert.NoError(t, err)

	ob := engine.getOrCreateOrderbook(1)
	assert.Len(t, ob.Asks, 1)
	assert.Len(t, ob.Bids, 0)

	// Create a buy order (Bid) for 50 @ 60
	bid, _ := generateTestOrder(t, 1, 50, 60, true)
	err = engine.ProcessOrder(bid)
	assert.NoError(t, err)

	// Wait for trade
	select {
	case trade := <-tradeQueue:
		assert.Equal(t, big.NewInt(50), trade.MatchAmount)
		assert.Equal(t, big.NewInt(50), trade.MatchPrice)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected a trade but got none")
	}

	assert.Len(t, ob.Asks, 1)
	assert.Equal(t, big.NewInt(50), ob.Asks[0].Amount, "Ask amount should be reduced to 50")
	assert.Len(t, ob.Bids, 0)
}

func TestEngine_PartialMatches(t *testing.T) {
	tradeQueue := make(chan types.Trade, 10)
	engine := NewEngine(tradeQueue)

	ask1, _ := generateTestOrder(t, 2, 40, 10, false)
	engine.ProcessOrder(ask1)

	ask2, _ := generateTestOrder(t, 2, 60, 15, false)
	engine.ProcessOrder(ask2)

	// Buy 70 @ 20. Should fill 40 @ 10, and 30 @ 15. 
	// Leaves ask2 with 30.
	bid, _ := generateTestOrder(t, 2, 70, 20, true)
	engine.ProcessOrder(bid)

	trades := make([]types.Trade, 0)
	for i := 0; i < 2; i++ {
		select {
		case trade := <-tradeQueue:
			trades = append(trades, trade)
		case <-time.After(1 * time.Second):
			t.Fatal("Expected 2 trades")
		}
	}

	assert.Equal(t, big.NewInt(40), trades[0].MatchAmount)
	assert.Equal(t, big.NewInt(10), trades[0].MatchPrice)

	assert.Equal(t, big.NewInt(30), trades[1].MatchAmount)
	assert.Equal(t, big.NewInt(15), trades[1].MatchPrice)

	ob := engine.getOrCreateOrderbook(2)
	assert.Len(t, ob.Asks, 1)
	assert.Equal(t, big.NewInt(30), ob.Asks[0].Amount)
}

func TestEngine_InvalidSignature(t *testing.T) {
	tradeQueue := make(chan types.Trade, 10)
	engine := NewEngine(tradeQueue)

	order, _ := generateTestOrder(t, 1, 100, 50, false)
	// Note: Strings in Go are immutable, so we can't just flip a bit like this in a string easily
	// We'll replace the first char with another valid hex char
	if order.Signature[0] == 'a' {
		order.Signature = "b" + order.Signature[1:]
	} else {
		order.Signature = "a" + order.Signature[1:]
	}

	err := engine.ProcessOrder(order)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid order signature")
}

func TestEngine_ZeroOrNegativeAmount(t *testing.T) {
	tradeQueue := make(chan types.Trade, 10)
	engine := NewEngine(tradeQueue)

	// Zero amount
	order1, _ := generateTestOrder(t, 1, 0, 50, false)
	err1 := engine.ProcessOrder(order1)
	assert.Error(t, err1)
	assert.Contains(t, err1.Error(), "amount must be > 0")

	// Negative amount (if manually forced into big.Int)
	order2, _ := generateTestOrder(t, 1, 10, 50, false)
	order2.Amount = big.NewInt(-10)
	err2 := engine.ProcessOrder(order2)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "amount must be > 0")

	// Zero price
	order3, _ := generateTestOrder(t, 1, 10, 0, false)
	err3 := engine.ProcessOrder(order3)
	assert.Error(t, err3)
	assert.Contains(t, err3.Error(), "price must be > 0")
}

