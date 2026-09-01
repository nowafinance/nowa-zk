package main

import (
	"context"
	"fmt"
	"log"
	"math/big"

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

	// 1. Resolve the depositor's account, keyed by their properly-compressed pubkey —
	// this MUST match the same encoding /account and /proof use, or the deposit
	// becomes invisible to the depositor's own future lookups. GetAccountID only
	// allocates an ID (a separate LevelDB key, not a Merkle leaf) — it doesn't touch
	// the tree, so this doesn't disturb oldRoot above.
	pubHex, err := compressPubKeyHex(event.PubKeyX, event.PubKeyY)
	if err != nil {
		// The L1 contract never validates curve membership at deposit() time, so a
		// malformed pubkey can theoretically reach here. Skip rather than silently
		// crediting a shared fallback bucket that would corrupt real accounting —
		// this deposit needs manual/admin recovery, which is safer than guessing.
		log.Printf("processDeposit: invalid pubkey in Deposit event, skipping: %v\n", err)
		return
	}
	accID, err := tree.GetAccountID(pubHex)
	if err != nil {
		log.Printf("processDeposit: failed to resolve account ID: %v\n", err)
		return
	}
	existing, err := tree.GetBalance(accID, event.TokenId)
	if err != nil {
		log.Printf("processDeposit: failed to load account: %v\n", err)
		return
	}
	isGenesis := existing == nil

	// 2. Snapshot the leaf's pre-deposit state. For a brand-new account this is the
	// tree's true (never-written) state — balance/nonce 0, IsGenesis=true, so the
	// circuit checks this leg's inclusion against the SMT's literal zero rather than
	// accountLeaf(index, pubX, pubY, 0, 0) (a real hash of the depositor's actual
	// pubkey, which a never-written leaf does NOT actually contain — conflating the
	// two was a real bug, confirmed live on Sepolia: batch #1's first-ever deposit
	// failed the TakerBase inclusion check for exactly this reason). See
	// prover/circuits/state_circuit.go's StateUpdate.IsGenesis doc comment.
	leafIndex := (accID * 256) + uint64(event.TokenId)
	path, bits := tree.GetPath(leafIndex)
	var pathStr [28]string
	for i := 0; i < 28; i++ {
		pathStr[i] = path[i].String()
	}

	balance := "0"
	nonce := uint64(0)
	if existing != nil {
		balance = existing.Balance.String()
		nonce = existing.Nonce
	}
	takerBaseUpdate := types.StateUpdate{
		Index:     leafIndex,
		Balance:   balance,
		Nonce:     nonce,
		IsGenesis: isGenesis,
		Path:      pathStr,
		PathBits:  bits,
	}

	// 3. Apply the deposit.
	acc := existing
	if acc == nil {
		acc = &types.BalanceState{
			AccountID: accID,
			TokenID:   event.TokenId,
			PubKeyX:   event.PubKeyX,
			PubKeyY:   event.PubKeyY,
			Balance:   big.NewInt(0),
			Nonce:     0,
		}
	}
	acc.Balance.Add(acc.Balance, event.Amount)
	// Notice: deposits don't increment nonce because they are driven by L1!
	// (Unless the protocol rules dictate otherwise, but usually they don't consume L2 nonces).
	if err := tree.SetBalance(acc); err != nil {
		log.Printf("processDeposit: failed to apply deposit: %v\n", err)
		return
	}
	newRoot := tree.Root().String()

	dummyUpdate := getEmptyStateUpdate(tree, 99)
	st := types.StateTransition{
		OpType:      3, // OpDeposit
		Amount:      event.Amount.String(),
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

