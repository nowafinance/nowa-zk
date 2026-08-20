package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/nowafinance/nowa-zk/sequencer/internal/bindings"
	"github.com/nowafinance/nowa-zk/sequencer/internal/batcher"
	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

// StartDepositWatcher connects to L1 and listens for Deposit events.
// It mints tokens in the L2 Merkle DB and adds OpDeposit transitions to the batch.
func StartDepositWatcher(rpcURL string, contractAddr string, tree *state.LevelDBMerkleTree, batch *batcher.Batcher) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Printf("DepositWatcher: Failed to connect to Ethereum: %v\n", err)
		return
	}

	addr := common.HexToAddress(contractAddr)
	contract, err := bindings.NewNowaRollup(addr, client)
	if err != nil {
		log.Printf("DepositWatcher: Failed to bind NowaRollup contract: %v\n", err)
		return
	}

	sink := make(chan *bindings.NowaRollupDeposit)
	sub, err := contract.WatchDeposit(&bind.WatchOpts{Context: context.Background()}, sink, nil, nil)
	if err != nil {
		log.Printf("DepositWatcher: Failed to watch Deposit events: %v\n", err)
		return
	}

	fmt.Println("DepositWatcher: Listening for L1 deposits...")

	for {
		select {
		case err := <-sub.Err():
			log.Printf("DepositWatcher: Subscription error: %v\n", err)
			return
		case event := <-sink:
			fmt.Printf("DepositWatcher: Received Deposit! User=%s, Token=%d, Amount=%s\n", event.User.Hex(), event.TokenId, event.Amount.String())
			
			// Process Deposit
			processDeposit(event, tree, batch)
		}
	}
}

func processDeposit(event *bindings.NowaRollupDeposit, tree *state.LevelDBMerkleTree, batch *batcher.Batcher) {
	oldRoot := tree.Root().String()

	// 1. Get or create the user account, keyed by their properly-compressed pubkey —
	// this MUST match the same encoding /account and /proof use, or the deposit
	// becomes invisible to the depositor's own future lookups.
	pubHex, err := compressPubKeyHex(event.PubKeyX, event.PubKeyY)
	if err != nil {
		// The L1 contract never validates curve membership at deposit() time, so a
		// malformed pubkey can theoretically reach here. Skip rather than silently
		// crediting a shared fallback bucket that would corrupt real accounting —
		// this deposit needs manual/admin recovery, which is safer than guessing.
		log.Printf("processDeposit: invalid pubkey in Deposit event, skipping: %v\n", err)
		return
	}
	acc, err := getOrCreateAccountForDeposit(tree, pubHex, event.PubKeyX, event.PubKeyY, event.TokenId)
	if err != nil {
		log.Printf("processDeposit: failed to load account: %v\n", err)
		return
	}

	// 2. Fetch path before update
	leafIndex := (acc.AccountID * 256) + uint64(event.TokenId)
	path, bits := tree.GetPath(leafIndex)
	var pathStr [28]string
	for i := 0; i < 28; i++ {
		pathStr[i] = path[i].String()
	}
	
	takerBaseUpdate := types.StateUpdate{
		Index:    leafIndex,
		Balance:  acc.Balance.String(),
		Nonce:    acc.Nonce,
		Path:     pathStr,
		PathBits: bits,
	}

	// 3. Apply deposit to balance
	acc.Balance.Add(acc.Balance, event.Amount)
	
	// Notice: deposits don't increment nonce because they are driven by L1!
	// (Unless the protocol rules dictate otherwise, but usually they don't consume L2 nonces).
	
	// Update tree
	tree.SetBalance(acc)

	newRoot := tree.Root().String()

	// Create a dummy maker state to fill the 4-path requirement (since it's a single deposit)
	dummyUpdate := getEmptyStateUpdate(tree, 99)

	st := types.StateTransition{
		OpType: 3, // OpDeposit
		Amount: event.Amount.String(),
		QuoteAmount: "0",

		// Maker is empty for deposit
		MakerPubKeyX: "0",
		MakerPubKeyY: "0",
		MakerSig:     "0x0000000000000000000000000000000000000000000000000000000000000000",
		MakerBase:    dummyUpdate,
		MakerQuote:   dummyUpdate,

		// Taker is the depositor
		TakerPubKeyX: event.PubKeyX.String(),
		TakerPubKeyY: event.PubKeyY.String(),
		TakerSig:     "0x0000000000000000000000000000000000000000000000000000000000000000",
		TakerBase:    takerBaseUpdate,
		TakerQuote:   dummyUpdate,
	}

	batch.AddTransition(st, oldRoot, newRoot)
}
