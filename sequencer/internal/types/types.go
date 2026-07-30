package types

import (
	"math/big"
)

// BalanceState represents a leaf in the flattened Sparse Merkle Tree (SMT).
// Leaf Index = (AccountID * 256) + TokenID
type BalanceState struct {
	AccountID uint64   // The unique ID of the user
	TokenID   uint32   // The ID of the token (0-255)
	PubKeyX   *big.Int // The user's Public Key (X coordinate)
	PubKeyY   *big.Int // The user's Public Key (Y coordinate)
	Balance   *big.Int // The user's current token balance
	Nonce     uint64   // The nonce for this specific token
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

// StateUpdate mirrors the GNARK SMT update structure
type StateUpdate struct {
	Index    uint64     `json:"index"`
	Balance  string     `json:"balance"`
	Nonce    uint64     `json:"nonce"`
	Path     [28]string `json:"path"`
	PathBits [28]uint64 `json:"path_bits"`
}

// StateTransition represents a single operation for the ZK Circuit.
type StateTransition struct {
	OpType int `json:"op_type"` // 0=Trade, 1=Transfer, 2=Withdrawal, 3=Deposit

	Amount      string `json:"amount"`
	QuoteAmount string `json:"quote_amount"`

	// Maker (Sender in Transfer/Withdraw)
	MakerPubKeyX string      `json:"maker_pub_key_x"`
	MakerPubKeyY string      `json:"maker_pub_key_y"`
	MakerSig     string      `json:"maker_sig"`
	MakerBase    StateUpdate `json:"maker_base"`
	MakerQuote   StateUpdate `json:"maker_quote"`

	// Taker (Receiver in Transfer/Deposit)
	TakerPubKeyX string      `json:"taker_pub_key_x"`
	TakerPubKeyY string      `json:"taker_pub_key_y"`
	TakerSig     string      `json:"taker_sig"`
	TakerBase    StateUpdate `json:"taker_base"`
	TakerQuote   StateUpdate `json:"taker_quote"`
}

// ZKBatch is the full payload sent to the Prover.
type ZKBatch struct {
	BatchID        uint64            `json:"batch_id"`
	OldRoot        string            `json:"old_root"`
	NewRoot        string            `json:"new_root"`
	WithdrawalHash string            `json:"withdrawal_hash"`
	DepositHash    string            `json:"deposit_hash"`
	Transitions    []StateTransition `json:"transitions"` // Array of 5 (BatchSize=5 in circuit)
}
