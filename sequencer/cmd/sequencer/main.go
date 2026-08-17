package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/nowafinance/nowa-zk/sequencer/internal/api"
	"github.com/nowafinance/nowa-zk/sequencer/internal/batcher"
	"github.com/nowafinance/nowa-zk/sequencer/internal/engine"
	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

func main() {
	fmt.Println("Starting Nowa-ZK Sequencer...")

	tree, err := state.NewLevelDBMerkleTree("./nowa_state_db", 28)
	if err != nil {
		log.Fatalf("Failed to initialize state DB: %v", err)
	}
	defer tree.Close()

	tradeQueue := make(chan types.Trade, 1000)
	eng := engine.NewEngine(tradeQueue)
	batch := batcher.NewBatcher()

	// Apply every real match into Merkle state + ZK batch (BatchSize=1 ⇒ seals immediately).
	go func() {
		for trade := range tradeQueue {
			fmt.Printf("Matched Trade: %s (Amount: %s, Price: %s)\n",
				trade.TradeID, trade.MatchAmount.String(), trade.MatchPrice.String())
			applyTrade(trade, tree, batch)
			if n := batch.GetBatchCount(); n > 0 {
				fmt.Printf("Sealed batches available: %d (latest ready for prover)\n", n)
			}
		}
	}()

	server := api.NewServer(eng, batch, tree)
	go func() {
		if err := server.Start(":8080"); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	rpcURL := os.Getenv("L1_RPC_URL")
	if rpcURL == "" {
		rpcURL = "http://localhost:8545"
	}
	contractAddr := os.Getenv("ROLLUP_CONTRACT_ADDRESS")
	if contractAddr == "" {
		home, _ := os.UserHomeDir()
		// optional: Makefile may export it
		_ = home
	}
	if contractAddr != "" {
		go StartDepositWatcher(rpcURL, contractAddr, tree, batch)
	} else {
		fmt.Println("No ROLLUP_CONTRACT_ADDRESS provided. Deposit watcher disabled.")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("\nShutting down Sequencer gracefully...")
}

func decodePubKeyXY(pubKeyHex string) (*big.Int, *big.Int, error) {
	clean := strings.TrimPrefix(pubKeyHex, "0x")
	raw, err := hex.DecodeString(clean)
	if err != nil {
		return nil, nil, err
	}
	var pk eddsa.PublicKey
	if _, err := pk.SetBytes(raw); err != nil {
		return nil, nil, err
	}
	x := new(big.Int)
	y := new(big.Int)
	pk.A.X.BigInt(x)
	pk.A.Y.BigInt(y)
	return x, y, nil
}

func getOrCreateBalance(tree *state.LevelDBMerkleTree, pubKeyHex string, tokenID uint32) (*types.BalanceState, error) {
	accID, err := tree.GetAccountID(pubKeyHex)
	if err != nil {
		return nil, err
	}

	acc, err := tree.GetBalance(accID, tokenID)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		pubX, pubY, err := decodePubKeyXY(pubKeyHex)
		if err != nil {
			pubX, pubY = big.NewInt(0), big.NewInt(0)
			fmt.Printf("getOrCreateBalance: pubkey decode failed (%v), using zeros\n", err)
		}
		acc = &types.BalanceState{
			AccountID: accID,
			TokenID:   tokenID,
			PubKeyX:   pubX,
			PubKeyY:   pubY,
			Balance:   new(big.Int).SetInt64(1000000), // lab credit until L1 deposit path is used
			Nonce:     0,
		}
		if err := tree.SetBalance(acc); err != nil {
			return nil, err
		}
	}
	return acc, nil
}

func getEmptyStateUpdate(tree *state.LevelDBMerkleTree, index uint64) types.StateUpdate {
	path, bits := tree.GetPath(index)
	var pathStr [28]string
	for i := 0; i < 28; i++ {
		pathStr[i] = path[i].String()
	}
	return types.StateUpdate{
		Index:    index,
		Balance:  "0",
		Nonce:    0,
		Path:     pathStr,
		PathBits: bits,
	}
}

func quoteTokenForBook(baseToken uint32) uint32 {
	if baseToken == 1 {
		return 2
	}
	return 1
}

func snapshotUpdate(tree *state.LevelDBMerkleTree, acc *types.BalanceState) types.StateUpdate {
	index := (acc.AccountID * 256) + uint64(acc.TokenID)
	path, bits := tree.GetPath(index)
	var pathStr [28]string
	for i := 0; i < 28; i++ {
		pathStr[i] = path[i].String()
	}
	return types.StateUpdate{
		Index: index, Balance: acc.Balance.String(), Nonce: acc.Nonce, Path: pathStr, PathBits: bits,
	}
}

// applyTrade orients the fill as circuit expects: maker = seller of base, taker = buyer paying quote.
// Partial fills: MatchAmount can be < resting size; remainder stays on the book with the same circuit auth.
func applyTrade(trade types.Trade, tree *state.LevelDBMerkleTree, batch *batcher.Batcher) {
	var sellOrder, buyOrder types.Order
	switch {
	case !trade.MakerOrder.IsBuy && trade.TakerOrder.IsBuy:
		sellOrder, buyOrder = trade.MakerOrder, trade.TakerOrder
	case trade.MakerOrder.IsBuy && !trade.TakerOrder.IsBuy:
		sellOrder, buyOrder = trade.TakerOrder, trade.MakerOrder
	default:
		fmt.Printf("applyTrade: expected one buy and one sell, skipping %s\n", trade.TradeID)
		return
	}

	if sellOrder.CircuitSignature == "" || buyOrder.CircuitSignature == "" {
		fmt.Printf("applyTrade: missing circuit_signature on orders, skipping %s\n", trade.TradeID)
		return
	}

	baseToken := sellOrder.TokenID
	quoteToken := quoteTokenForBook(baseToken)
	if sellOrder.TokenID != buyOrder.TokenID {
		// Rare dual-token path from unit tests: buyer sells their TokenID as quote.
		quoteToken = buyOrder.TokenID
		baseToken = sellOrder.TokenID
	}

	matchAmount := trade.MatchAmount
	matchQuote := new(big.Int).Mul(trade.MatchAmount, trade.MatchPrice)

	sellerBase, err := getOrCreateBalance(tree, sellOrder.MakerAddress, baseToken)
	if err != nil {
		fmt.Printf("applyTrade: seller base: %v\n", err)
		return
	}
	sellerQuote, err := getOrCreateBalance(tree, sellOrder.MakerAddress, quoteToken)
	if err != nil {
		fmt.Printf("applyTrade: seller quote: %v\n", err)
		return
	}
	buyerBase, err := getOrCreateBalance(tree, buyOrder.MakerAddress, baseToken)
	if err != nil {
		fmt.Printf("applyTrade: buyer base: %v\n", err)
		return
	}
	buyerQuote, err := getOrCreateBalance(tree, buyOrder.MakerAddress, quoteToken)
	if err != nil {
		fmt.Printf("applyTrade: buyer quote: %v\n", err)
		return
	}

	if sellerBase.Balance.Cmp(matchAmount) < 0 {
		fmt.Printf("applyTrade: insufficient seller base, skipping %s\n", trade.TradeID)
		return
	}
	if buyerQuote.Balance.Cmp(matchQuote) < 0 {
		fmt.Printf("applyTrade: insufficient buyer quote, skipping %s\n", trade.TradeID)
		return
	}

	oldRoot := tree.Root().String()

	// Paths must be taken in the same order the circuit updates leaves.
	makerBaseUpdate := snapshotUpdate(tree, sellerBase)
	sellerBase.Balance.Sub(sellerBase.Balance, matchAmount)
	if err := tree.SetBalance(sellerBase); err != nil {
		fmt.Printf("applyTrade: set seller base: %v\n", err)
		return
	}

	makerQuoteUpdate := snapshotUpdate(tree, sellerQuote)
	sellerQuote.Balance.Add(sellerQuote.Balance, matchQuote)
	if err := tree.SetBalance(sellerQuote); err != nil {
		fmt.Printf("applyTrade: set seller quote: %v\n", err)
		return
	}

	takerBaseUpdate := snapshotUpdate(tree, buyerBase)
	buyerBase.Balance.Add(buyerBase.Balance, matchAmount)
	if err := tree.SetBalance(buyerBase); err != nil {
		fmt.Printf("applyTrade: set buyer base: %v\n", err)
		return
	}

	takerQuoteUpdate := snapshotUpdate(tree, buyerQuote)
	buyerQuote.Balance.Sub(buyerQuote.Balance, matchQuote)
	if err := tree.SetBalance(buyerQuote); err != nil {
		fmt.Printf("applyTrade: set buyer quote: %v\n", err)
		return
	}

	newRoot := tree.Root().String()

	st := types.StateTransition{
		OpType:      0,
		Amount:      matchAmount.String(),
		QuoteAmount: matchQuote.String(),

		MakerPubKeyX: sellerBase.PubKeyX.String(),
		MakerPubKeyY: sellerBase.PubKeyY.String(),
		MakerSig:     sellOrder.CircuitSignature,
		MakerBase:    makerBaseUpdate,
		MakerQuote:   makerQuoteUpdate,

		TakerPubKeyX: buyerBase.PubKeyX.String(),
		TakerPubKeyY: buyerBase.PubKeyY.String(),
		TakerSig:     buyOrder.CircuitSignature,
		TakerBase:    takerBaseUpdate,
		TakerQuote:   takerQuoteUpdate,
	}

	batch.AddTransition(st, oldRoot, newRoot)
	fmt.Printf("applyTrade: applied fill %s amount=%s quote=%s (sealed batches=%d)\n",
		trade.TradeID, matchAmount.String(), matchQuote.String(), batch.GetBatchCount())
}
