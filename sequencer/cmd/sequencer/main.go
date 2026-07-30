package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"math/big"
	"strings"
	"time"

	"github.com/nowafinance/nowa-zk/sequencer/internal/api"
	"github.com/nowafinance/nowa-zk/sequencer/internal/batcher"
	"github.com/nowafinance/nowa-zk/sequencer/internal/engine"
	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

func main() {
	fmt.Println("Starting Nowa-ZK Sequencer...")

	// 1. Initialize State DB (LevelDB SMT)
	tree, err := state.NewLevelDBMerkleTree("./nowa_state_db", 28)
	if err != nil {
		log.Fatalf("Failed to initialize state DB: %v", err)
	}
	defer tree.Close()

	// 2. Start Matching Engine
	tradeQueue := make(chan types.Trade, 1000)
	eng := engine.NewEngine(tradeQueue)

	batch := batcher.NewBatcher()

	// Background worker to process trades and update state DB
	go func() {
		for trade := range tradeQueue {
			fmt.Printf("Matched Trade: %s (Amount: %s, Price: %s)\n", trade.TradeID, trade.MatchAmount.String(), trade.MatchPrice.String())
			applyTrade(trade, tree, batch)
		}
	}()

	// Background worker to auto-pad batches with dummy trades every 10 seconds if there are pending trades
	go func() {
		for {
			time.Sleep(10 * time.Second)
			current := batch.GetCurrentBatchSize()
			if current > 0 && current < batcher.BatchSize {
				fmt.Printf("Auto-padding batch with %d dummy trades...\n", batcher.BatchSize-current)
				for i := current; i < batcher.BatchSize; i++ {
					applyDummyTrade(tree, batch)
				}
			}
		}
	}()

	// Start API Server (inject batcher as well)
	server := api.NewServer(eng, batch)
	go func() {
		if err := server.Start(":8080"); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// 4. Start L1 Deposit Watcher
	// Use environment variables for L1 connection
	rpcURL := os.Getenv("L1_RPC_URL")
	if rpcURL == "" { rpcURL = "http://localhost:8545" }
	contractAddr := os.Getenv("ROLLUP_CONTRACT_ADDRESS")
	if contractAddr != "" {
		go StartDepositWatcher(rpcURL, contractAddr, tree, batch)
	} else {
		fmt.Println("No ROLLUP_CONTRACT_ADDRESS provided. Deposit watcher disabled.")
	}

	// Wait for termination signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("\nShutting down Sequencer gracefully...")
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
		// Decode pubkey 
		cleanHex := strings.TrimPrefix(pubKeyHex, "0x")
		pubX, _ := new(big.Int).SetString(cleanHex[:len(cleanHex)/2], 16)
		pubY, _ := new(big.Int).SetString(cleanHex[len(cleanHex)/2:], 16)
		if pubX == nil { pubX = big.NewInt(0) }
		if pubY == nil { pubY = big.NewInt(0) }

		acc = &types.BalanceState{
			AccountID: accID,
			TokenID:   tokenID,
			PubKeyX:   pubX,
			PubKeyY:   pubY,
			Balance:   new(big.Int).SetInt64(1000000), // Give new balances 1 million tokens for testing!
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

func applyTrade(trade types.Trade, tree *state.LevelDBMerkleTree, batch *batcher.Batcher) {
	oldRoot := tree.Root().String()

	// In a real system, the maker is selling TokenA for TokenB, and the taker is selling TokenB for TokenA.
	makerBaseToken := trade.MakerOrder.TokenID // Token being sold by Maker
	takerBaseToken := trade.TakerOrder.TokenID // Token being sold by Taker (Maker's Quote token)

	// Maker State
	makerBaseAcc, _ := getOrCreateBalance(tree, trade.MakerOrder.MakerAddress, makerBaseToken)
	makerQuoteAcc, _ := getOrCreateBalance(tree, trade.MakerOrder.MakerAddress, takerBaseToken)

	// Taker State
	takerBaseAcc, _ := getOrCreateBalance(tree, trade.TakerOrder.MakerAddress, takerBaseToken)
	takerQuoteAcc, _ := getOrCreateBalance(tree, trade.TakerOrder.MakerAddress, makerBaseToken)

	// 1. Fetch paths before updates
	makerBaseIndex := (makerBaseAcc.AccountID * 256) + uint64(makerBaseAcc.TokenID)
	makerBasePath, makerBaseBits := tree.GetPath(makerBaseIndex)
	var makerBasePathStr [28]string
	for i := 0; i < 28; i++ { makerBasePathStr[i] = makerBasePath[i].String() }
	makerBaseUpdate := types.StateUpdate{
		Index: makerBaseIndex, Balance: makerBaseAcc.Balance.String(), Nonce: makerBaseAcc.Nonce, Path: makerBasePathStr, PathBits: makerBaseBits,
	}

	// 2. Perform updates
	makerBaseAcc.Balance.Sub(makerBaseAcc.Balance, trade.MatchAmount)
	makerBaseAcc.Nonce++
	tree.SetBalance(makerBaseAcc)

	makerQuoteIndex := (makerQuoteAcc.AccountID * 256) + uint64(makerQuoteAcc.TokenID)
	makerQuotePath, makerQuoteBits := tree.GetPath(makerQuoteIndex)
	var makerQuotePathStr [28]string
	for i := 0; i < 28; i++ { makerQuotePathStr[i] = makerQuotePath[i].String() }
	makerQuoteUpdate := types.StateUpdate{
		Index: makerQuoteIndex, Balance: makerQuoteAcc.Balance.String(), Nonce: makerQuoteAcc.Nonce, Path: makerQuotePathStr, PathBits: makerQuoteBits,
	}

	matchQuoteAmount := new(big.Int).Mul(trade.MatchAmount, trade.MatchPrice)
	makerQuoteAcc.Balance.Add(makerQuoteAcc.Balance, matchQuoteAmount)
	tree.SetBalance(makerQuoteAcc)

	takerBaseIndex := (takerBaseAcc.AccountID * 256) + uint64(takerBaseAcc.TokenID)
	takerBasePath, takerBaseBits := tree.GetPath(takerBaseIndex)
	var takerBasePathStr [28]string
	for i := 0; i < 28; i++ { takerBasePathStr[i] = takerBasePath[i].String() }
	takerBaseUpdate := types.StateUpdate{
		Index: takerBaseIndex, Balance: takerBaseAcc.Balance.String(), Nonce: takerBaseAcc.Nonce, Path: takerBasePathStr, PathBits: takerBaseBits,
	}

	takerBaseAcc.Balance.Sub(takerBaseAcc.Balance, matchQuoteAmount)
	takerBaseAcc.Nonce++
	tree.SetBalance(takerBaseAcc)

	takerQuoteIndex := (takerQuoteAcc.AccountID * 256) + uint64(takerQuoteAcc.TokenID)
	takerQuotePath, takerQuoteBits := tree.GetPath(takerQuoteIndex)
	var takerQuotePathStr [28]string
	for i := 0; i < 28; i++ { takerQuotePathStr[i] = takerQuotePath[i].String() }
	takerQuoteUpdate := types.StateUpdate{
		Index: takerQuoteIndex, Balance: takerQuoteAcc.Balance.String(), Nonce: takerQuoteAcc.Nonce, Path: takerQuotePathStr, PathBits: takerQuoteBits,
	}

	takerQuoteAcc.Balance.Add(takerQuoteAcc.Balance, trade.MatchAmount)
	tree.SetBalance(takerQuoteAcc)

	newRoot := tree.Root().String()

	st := types.StateTransition{
		OpType:       0, // OpTrade
		Amount:       trade.MatchAmount.String(),
		QuoteAmount:  matchQuoteAmount.String(),

		MakerPubKeyX: makerBaseAcc.PubKeyX.String(),
		MakerPubKeyY: makerBaseAcc.PubKeyY.String(),
		MakerSig:     trade.MakerOrder.Signature,
		MakerBase:    makerBaseUpdate,
		MakerQuote:   makerQuoteUpdate,

		TakerPubKeyX: takerBaseAcc.PubKeyX.String(),
		TakerPubKeyY: takerBaseAcc.PubKeyY.String(),
		TakerSig:     trade.TakerOrder.Signature,
		TakerBase:    takerBaseUpdate,
		TakerQuote:   takerQuoteUpdate,
	}

	batch.AddTransition(st, oldRoot, newRoot)
}

func applyDummyTrade(tree *state.LevelDBMerkleTree, batch *batcher.Batcher) {
	oldRoot := tree.Root().String()

	acc, _ := getOrCreateBalance(tree, "0x0000000000000000000000000000000000000000", 0)
	leafIndex := (acc.AccountID * 256) + uint64(acc.TokenID)
	
	makerBaseUpdate := getEmptyStateUpdate(tree, leafIndex)
	makerBaseUpdate.Balance = acc.Balance.String()
	makerBaseUpdate.Nonce = acc.Nonce

	// A dummy transfer is just OpType=1, Amount=0. We use makerBaseUpdate.
	st := types.StateTransition{
		OpType: 1, // OpTransfer
		Amount: "0",
		QuoteAmount: "0",

		MakerPubKeyX: acc.PubKeyX.String(),
		MakerPubKeyY: acc.PubKeyY.String(),
		MakerSig:     "0x0000000000000000000000000000000000000000000000000000000000000000",
		MakerBase:    makerBaseUpdate,
		MakerQuote:   getEmptyStateUpdate(tree, 99),

		TakerPubKeyX: acc.PubKeyX.String(),
		TakerPubKeyY: acc.PubKeyY.String(),
		TakerSig:     "0x0000000000000000000000000000000000000000000000000000000000000000",
		TakerBase:    getEmptyStateUpdate(tree, leafIndex),
		TakerQuote:   getEmptyStateUpdate(tree, 99),
	}

	batch.AddTransition(st, oldRoot, oldRoot) // root doesn't change
}
