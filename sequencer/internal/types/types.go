package types

import (
	"math/big"
)

// AccountState represents a leaf in the Sparse Merkle Tree (SMT).
// This is exactly how the Go Sequencer handles balances without needing a blockchain!
type AccountState struct {
	PubKeyX *big.Int // The user's Public Key (X coordinate)
	PubKeyY *big.Int // The user's Public Key (Y coordinate)
	Balance *big.Int // The user's current token balance (e.g. USDC)
	Nonce   uint64   // To prevent replay attacks on trades
}

// Order represents a user's signed intent to trade.
type Order struct {
	MakerAddress string   `json:"maker_address"`
	TokenID      uint32   `json:"token_id"` // E.g., 1 for USDC, 2 for ETH
	Amount       *big.Int `json:"amount"`
	Price        *big.Int `json:"price"`
	IsBuy        bool     `json:"is_buy"`
	Nonce        uint64   `json:"nonce"`
	
	// EdDSA Signature (64 bytes in hex)
	Signature string `json:"signature"`
}

// Trade represents a matched buy and sell order.
// This is what gets sent to the Prover.
type Trade struct {
	TradeID     string `json:"trade_id"`
	MakerOrder  Order  `json:"maker_order"`
	TakerOrder  Order  `json:"taker_order"`
	MatchPrice  *big.Int `json:"match_price"`
	MatchAmount *big.Int `json:"match_amount"`
}

// Batch represents a group of exactly 25 matched trades (matches our Gnark circuit).
type Batch struct {
	BatchID uint64  `json:"batch_id"`
	Trades  []Trade `json:"trades"` // Must be exactly length 25
}
