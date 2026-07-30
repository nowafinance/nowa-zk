package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nowafinance/nowa-zk/sequencer/internal/api"
	"github.com/nowafinance/nowa-zk/sequencer/internal/engine"
	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

func main() {
	fmt.Println("Starting Nowa-ZK Sequencer...")

	// 1. Initialize State DB (LevelDB SMT)
	tree, err := state.NewLevelDBMerkleTree("./nowa_state_db", 20)
	if err != nil {
		log.Fatalf("Failed to initialize state DB: %v", err)
	}
	defer tree.Close()

	// 2. Start Matching Engine
	tradeQueue := make(chan types.Trade, 1000)
	eng := engine.NewEngine(tradeQueue)

	// Background worker to process trades and update state DB (Batching logic will go here)
	go func() {
		for trade := range tradeQueue {
			fmt.Printf("Matched Trade: %s (Amount: %s, Price: %s)\n", trade.TradeID, trade.MatchAmount.String(), trade.MatchPrice.String())
			// Here we will eventually apply the trade to the tree, increment nonces, and prepare batches of 25 for the Prover!
		}
	}()

	// 3. Start API Server
	server := api.NewServer(eng)
	go func() {
		if err := server.Start(":8080"); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for termination signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Println("\nShutting down Sequencer gracefully...")
}
