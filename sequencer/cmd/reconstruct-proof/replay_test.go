package main

import (
	"math/big"
	"os"
	"testing"

	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

func newTestTree(t *testing.T) *state.LevelDBMerkleTree {
	t.Helper()
	dir, err := os.MkdirTemp("", "reconstruct-proof-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	tree, err := state.NewLevelDBMerkleTree(dir, 28)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	return tree
}

func balance(t *testing.T, tree *state.LevelDBMerkleTree, accountID uint64, tokenID uint32) *types.BalanceState {
	t.Helper()
	acc, err := tree.GetBalance(accountID, tokenID)
	if err != nil {
		t.Fatal(err)
	}
	if acc == nil {
		t.Fatalf("no balance at account %d token %d", accountID, tokenID)
	}
	return acc
}

// seed writes an initial leaf directly, mirroring what a real prior transition
// would have left behind — every replay test starts from a known baseline.
func seed(t *testing.T, tree *state.LevelDBMerkleTree, accountID uint64, tokenID uint32, pubX, pubY, bal int64, nonce uint64) {
	t.Helper()
	err := tree.SetBalance(&types.BalanceState{
		AccountID: accountID, TokenID: tokenID,
		PubKeyX: big.NewInt(pubX), PubKeyY: big.NewInt(pubY),
		Balance: big.NewInt(bal), Nonce: nonce,
	})
	if err != nil {
		t.Fatal(err)
	}
}

// snapshot builds the StateUpdate a real transition would carry for this leaf's
// CURRENT (pre-transition) state — matches applyTrade/processDeposit's snapshotUpdate.
func snapshot(t *testing.T, tree *state.LevelDBMerkleTree, accountID uint64, tokenID uint32) types.StateUpdate {
	t.Helper()
	acc := balance(t, tree, accountID, tokenID)
	index := accountID*256 + uint64(tokenID)
	path, bits := tree.GetPath(index)
	var pathStr [28]string
	for i := range pathStr {
		pathStr[i] = path[i].String()
	}
	return types.StateUpdate{Index: index, Balance: acc.Balance.String(), Nonce: acc.Nonce, Path: pathStr, PathBits: bits}
}

func TestReplayTransition_Trade(t *testing.T) {
	tree := newTestTree(t)
	// Seller (maker): account 1, base(token1)=1000, quote(token2)=0
	seed(t, tree, 1, 1, 11, 22, 1000, 3)
	seed(t, tree, 1, 2, 11, 22, 0, 0)
	// Buyer (taker): account 2, base(token1)=0, quote(token2)=5000
	seed(t, tree, 2, 1, 33, 44, 0, 0)
	seed(t, tree, 2, 2, 33, 44, 5000, 7)

	st := types.StateTransition{
		OpType: opTrade, Amount: "100", QuoteAmount: "2500",
		MakerPubKeyX: "11", MakerPubKeyY: "22",
		MakerBase: snapshot(t, tree, 1, 1), MakerQuote: snapshot(t, tree, 1, 2),
		TakerPubKeyX: "33", TakerPubKeyY: "44",
		TakerBase: snapshot(t, tree, 2, 1), TakerQuote: snapshot(t, tree, 2, 2),
	}

	if err := replayTransition(tree, st); err != nil {
		t.Fatal(err)
	}

	makerBase := balance(t, tree, 1, 1)
	if makerBase.Balance.String() != "900" {
		t.Errorf("maker base balance: got %s, want 900", makerBase.Balance)
	}
	if makerBase.Nonce != 3 { // trades do NOT bump nonce
		t.Errorf("maker base nonce: got %d, want 3 (unchanged)", makerBase.Nonce)
	}
	makerQuote := balance(t, tree, 1, 2)
	if makerQuote.Balance.String() != "2500" {
		t.Errorf("maker quote balance: got %s, want 2500", makerQuote.Balance)
	}
	takerBase := balance(t, tree, 2, 1)
	if takerBase.Balance.String() != "100" {
		t.Errorf("taker base balance: got %s, want 100", takerBase.Balance)
	}
	takerQuote := balance(t, tree, 2, 2)
	if takerQuote.Balance.String() != "2500" {
		t.Errorf("taker quote balance: got %s, want 2500 (5000-2500)", takerQuote.Balance)
	}
	if takerQuote.Nonce != 7 {
		t.Errorf("taker quote nonce: got %d, want 7 (unchanged)", takerQuote.Nonce)
	}
}

func TestReplayTransition_Transfer(t *testing.T) {
	tree := newTestTree(t)
	seed(t, tree, 1, 1, 11, 22, 500, 2)
	seed(t, tree, 2, 1, 33, 44, 10, 0)

	st := types.StateTransition{
		OpType: opTransfer, Amount: "50", QuoteAmount: "0",
		MakerPubKeyX: "11", MakerPubKeyY: "22",
		MakerBase: snapshot(t, tree, 1, 1), MakerQuote: snapshot(t, tree, 1, 1), // quote leg inactive, value irrelevant
		TakerPubKeyX: "33", TakerPubKeyY: "44",
		TakerBase: snapshot(t, tree, 2, 1), TakerQuote: snapshot(t, tree, 2, 1),
	}
	if err := replayTransition(tree, st); err != nil {
		t.Fatal(err)
	}

	sender := balance(t, tree, 1, 1)
	if sender.Balance.String() != "450" || sender.Nonce != 3 {
		t.Errorf("sender: got balance=%s nonce=%d, want balance=450 nonce=3 (transfers DO bump nonce)", sender.Balance, sender.Nonce)
	}
	receiver := balance(t, tree, 2, 1)
	if receiver.Balance.String() != "60" || receiver.Nonce != 0 {
		t.Errorf("receiver: got balance=%s nonce=%d, want balance=60 nonce=0 (unchanged)", receiver.Balance, receiver.Nonce)
	}
}

func TestReplayTransition_Withdrawal(t *testing.T) {
	tree := newTestTree(t)
	seed(t, tree, 1, 1, 11, 22, 300, 5)

	st := types.StateTransition{
		OpType: opWithdrawal, Amount: "300", QuoteAmount: "0",
		MakerPubKeyX: "11", MakerPubKeyY: "22",
		MakerBase: snapshot(t, tree, 1, 1), MakerQuote: snapshot(t, tree, 1, 1),
		// Taker legs are inactive for withdrawals — deliberately left as zero values.
	}
	if err := replayTransition(tree, st); err != nil {
		t.Fatal(err)
	}
	withdrawer := balance(t, tree, 1, 1)
	if withdrawer.Balance.Sign() != 0 || withdrawer.Nonce != 6 {
		t.Errorf("withdrawer: got balance=%s nonce=%d, want balance=0 nonce=6", withdrawer.Balance, withdrawer.Nonce)
	}
}

func TestReplayTransition_Deposit(t *testing.T) {
	tree := newTestTree(t)
	seed(t, tree, 5, 1, 77, 88, 0, 0)

	st := types.StateTransition{
		OpType: opDeposit, Amount: "1000", QuoteAmount: "0",
		MakerPubKeyX: "0", MakerPubKeyY: "0", // dummy — inactive for deposits
		MakerBase: types.StateUpdate{Index: 99, Balance: "0", Nonce: 0},
		MakerQuote: types.StateUpdate{Index: 99, Balance: "0", Nonce: 0},
		TakerPubKeyX: "77", TakerPubKeyY: "88",
		TakerBase: snapshot(t, tree, 5, 1), TakerQuote: snapshot(t, tree, 5, 1),
	}
	if err := replayTransition(tree, st); err != nil {
		t.Fatal(err)
	}
	depositor := balance(t, tree, 5, 1)
	if depositor.Balance.String() != "1000" || depositor.Nonce != 0 {
		t.Errorf("depositor: got balance=%s nonce=%d, want balance=1000 nonce=0 (deposits never bump nonce)", depositor.Balance, depositor.Nonce)
	}
}

func TestReplayTransition_RejectsNegativeBalance(t *testing.T) {
	tree := newTestTree(t)
	seed(t, tree, 1, 1, 11, 22, 10, 0) // only 10, trying to debit 100

	st := types.StateTransition{
		OpType: opWithdrawal, Amount: "100", QuoteAmount: "0",
		MakerPubKeyX: "11", MakerPubKeyY: "22",
		MakerBase: snapshot(t, tree, 1, 1), MakerQuote: snapshot(t, tree, 1, 1),
	}
	if err := replayTransition(tree, st); err == nil {
		t.Fatal("expected an error replaying a transition that would produce a negative balance")
	}
}
