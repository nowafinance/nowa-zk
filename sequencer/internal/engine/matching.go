package engine

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

// Orderbook maintains the limit orders for a specific token pair (BaseToken vs USDC).
type Orderbook struct {
	TokenID uint32
	Bids    []*types.Order // Buy orders, sorted descending by price
	Asks    []*types.Order // Sell orders, sorted ascending by price
	mu      sync.Mutex
}

// Engine manages orderbooks for all tokens.
type Engine struct {
	Orderbooks map[uint32]*Orderbook
	TradeQueue chan types.Trade // Channel to send matched trades to the batcher
	mu         sync.RWMutex
}

func NewEngine(tradeQueue chan types.Trade) *Engine {
	return &Engine{
		Orderbooks: make(map[uint32]*Orderbook),
		TradeQueue: tradeQueue,
	}
}

func (e *Engine) getOrCreateOrderbook(tokenID uint32) *Orderbook {
	e.mu.Lock()
	defer e.mu.Unlock()

	if ob, exists := e.Orderbooks[tokenID]; exists {
		return ob
	}
	ob := &Orderbook{TokenID: tokenID}
	e.Orderbooks[tokenID] = ob
	return ob
}

// ProcessOrder receives an incoming order and matches it or adds it to the book.
func (e *Engine) ProcessOrder(order *types.Order) error {
	// 0. Validate amounts
	if order.Amount.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("invalid order: amount must be > 0")
	}
	if order.Price.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("invalid order: price must be > 0")
	}

	// 1. Verify Signature
	valid, err := VerifyEdDSASignature(order.MakerAddress, order.TokenID, order.Amount, order.Price, order.IsBuy, order.Nonce, order.Signature)
	if err != nil || !valid {
		return fmt.Errorf("invalid order signature: %v", err)
	}

	ob := e.getOrCreateOrderbook(order.TokenID)
	ob.mu.Lock()
	defer ob.mu.Unlock()

	remainingAmount := new(big.Int).Set(order.Amount)

	// 2. Match Order
	if order.IsBuy {
		remainingAmount = e.matchBids(ob, order, remainingAmount)
		if remainingAmount.Cmp(big.NewInt(0)) > 0 {
			// Add remaining to Bids
			order.Amount = remainingAmount
			e.insertBid(ob, order)
		}
	} else {
		remainingAmount = e.matchAsks(ob, order, remainingAmount)
		if remainingAmount.Cmp(big.NewInt(0)) > 0 {
			// Add remaining to Asks
			order.Amount = remainingAmount
			e.insertAsk(ob, order)
		}
	}

	return nil
}

// matchBids matches an incoming BUY order against existing ASKS.
func (e *Engine) matchBids(ob *Orderbook, incoming *types.Order, remainingAmount *big.Int) *big.Int {
	for len(ob.Asks) > 0 && remainingAmount.Cmp(big.NewInt(0)) > 0 {
		bestAsk := ob.Asks[0]

		// If incoming buy price < best ask price, no match
		if incoming.Price.Cmp(bestAsk.Price) < 0 {
			break
		}

		matchAmount := new(big.Int)
		if remainingAmount.Cmp(bestAsk.Amount) >= 0 {
			matchAmount.Set(bestAsk.Amount)
		} else {
			matchAmount.Set(remainingAmount)
		}

		// Emit Trade
		trade := types.Trade{
			TradeID:     fmt.Sprintf("tr_%d", time.Now().UnixNano()),
			MakerOrder:  *bestAsk,
			TakerOrder:  *incoming,
			MatchPrice:  bestAsk.Price,
			MatchAmount: matchAmount,
		}
		e.TradeQueue <- trade

		// Update Amounts
		remainingAmount.Sub(remainingAmount, matchAmount)
		bestAsk.Amount.Sub(bestAsk.Amount, matchAmount)

		// Remove filled ask
		if bestAsk.Amount.Cmp(big.NewInt(0)) == 0 {
			ob.Asks = ob.Asks[1:]
		}
	}
	return remainingAmount
}

// matchAsks matches an incoming SELL order against existing BIDS.
func (e *Engine) matchAsks(ob *Orderbook, incoming *types.Order, remainingAmount *big.Int) *big.Int {
	for len(ob.Bids) > 0 && remainingAmount.Cmp(big.NewInt(0)) > 0 {
		bestBid := ob.Bids[0]

		// If incoming sell price > best bid price, no match
		if incoming.Price.Cmp(bestBid.Price) > 0 {
			break
		}

		matchAmount := new(big.Int)
		if remainingAmount.Cmp(bestBid.Amount) >= 0 {
			matchAmount.Set(bestBid.Amount)
		} else {
			matchAmount.Set(remainingAmount)
		}

		// Emit Trade
		trade := types.Trade{
			TradeID:     fmt.Sprintf("tr_%d", time.Now().UnixNano()),
			MakerOrder:  *bestBid,
			TakerOrder:  *incoming,
			MatchPrice:  bestBid.Price,
			MatchAmount: matchAmount,
		}
		e.TradeQueue <- trade

		// Update Amounts
		remainingAmount.Sub(remainingAmount, matchAmount)
		bestBid.Amount.Sub(bestBid.Amount, matchAmount)

		// Remove filled bid
		if bestBid.Amount.Cmp(big.NewInt(0)) == 0 {
			ob.Bids = ob.Bids[1:]
		}
	}
	return remainingAmount
}

func (e *Engine) insertBid(ob *Orderbook, order *types.Order) {
	// Insert maintaining descending price order
	insertIdx := 0
	for i, bid := range ob.Bids {
		if order.Price.Cmp(bid.Price) > 0 {
			break
		}
		insertIdx = i + 1
	}
	
	if insertIdx == len(ob.Bids) {
		ob.Bids = append(ob.Bids, order)
	} else {
		ob.Bids = append(ob.Bids, nil)
		copy(ob.Bids[insertIdx+1:], ob.Bids[insertIdx:])
		ob.Bids[insertIdx] = order
	}
}

func (e *Engine) insertAsk(ob *Orderbook, order *types.Order) {
	// Insert maintaining ascending price order
	insertIdx := 0
	for i, ask := range ob.Asks {
		if order.Price.Cmp(ask.Price) < 0 {
			break
		}
		insertIdx = i + 1
	}
	
	if insertIdx == len(ob.Asks) {
		ob.Asks = append(ob.Asks, order)
	} else {
		ob.Asks = append(ob.Asks, nil)
		copy(ob.Asks[insertIdx+1:], ob.Asks[insertIdx:])
		ob.Asks[insertIdx] = order
	}
}
