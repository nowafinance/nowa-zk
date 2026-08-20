package api

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/nowafinance/nowa-zk/sequencer/internal/batcher"
	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
)

// newTestPubKeyHex generates a real, curve-valid compressed EdDSA pubkey — openBalance
// runs it through decodePubKeyXY, which (correctly) rejects anything else, unlike a
// fixture such as "0x1234".
func newTestPubKeyHex(t *testing.T) string {
	t.Helper()
	priv, err := eddsa.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return "0x" + hex.EncodeToString(priv.PublicKey.Bytes())
}

func newAccountTestTree(t *testing.T) *state.LevelDBMerkleTree {
	t.Helper()
	dir, err := os.MkdirTemp("", "nowa-account-test-*")
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

// TestOpenBalance_TracksLabCreditAsTransition is the core regression test for the
// account-onboarding bug: openBalance used to write the lab-credit leaf directly to
// the tree with no corresponding StateTransition, invisible to the batcher. This
// asserts the credit now shows up as a real, batcher-sealed OpDeposit transition
// whose OldRoot/NewRoot bracket the exact root change the write produced.
func TestOpenBalance_TracksLabCreditAsTransition(t *testing.T) {
	tree := newAccountTestTree(t)
	b := batcher.NewBatcher()

	rootBefore := tree.Root().String()

	acc, err := openBalance(tree, b, newTestPubKeyHex(t), 1)
	if err != nil {
		t.Fatalf("openBalance: %v", err)
	}
	if acc.Balance.String() != "1000000" {
		t.Fatalf("balance: got %s, want 1000000", acc.Balance)
	}

	rootAfter := tree.Root().String()
	if rootAfter == rootBefore {
		t.Fatal("tree root did not change after crediting a new account")
	}

	sealed := b.GetLatestBatch()
	if sealed == nil {
		t.Fatal("expected a sealed batch after the lab-credit transition, got none")
	}
	if len(sealed.Transitions) != 1 {
		t.Fatalf("expected exactly 1 transition, got %d", len(sealed.Transitions))
	}
	tr := sealed.Transitions[0]
	if tr.OpType != batcher.OpDeposit {
		t.Fatalf("OpType: got %d, want OpDeposit(%d)", tr.OpType, batcher.OpDeposit)
	}
	if tr.Amount != "1000000" {
		t.Fatalf("Amount: got %s, want 1000000", tr.Amount)
	}
	wantIndex := (acc.AccountID * 256) + uint64(acc.TokenID)
	if tr.TakerBase.Index != wantIndex {
		t.Fatalf("TakerBase.Index: got %d, want %d", tr.TakerBase.Index, wantIndex)
	}
	if sealed.OldRoot != rootBefore {
		t.Fatalf("batch OldRoot: got %s, want %s (tree root before the credit)", sealed.OldRoot, rootBefore)
	}
	if sealed.NewRoot != rootAfter {
		t.Fatalf("batch NewRoot: got %s, want %s (tree root after the credit)", sealed.NewRoot, rootAfter)
	}
}

// TestOpenBalance_SecondCallDoesNotDuplicateCredit confirms an already-funded account
// is returned as-is, with no second transition minted.
func TestOpenBalance_SecondCallDoesNotDuplicateCredit(t *testing.T) {
	tree := newAccountTestTree(t)
	b := batcher.NewBatcher()

	pub := newTestPubKeyHex(t)
	first, err := openBalance(tree, b, pub, 1)
	if err != nil {
		t.Fatalf("first openBalance: %v", err)
	}
	second, err := openBalance(tree, b, pub, 1)
	if err != nil {
		t.Fatalf("second openBalance: %v", err)
	}
	if second.Balance.String() != first.Balance.String() {
		t.Fatalf("second call changed balance: got %s, want %s (unchanged)", second.Balance, first.Balance)
	}
	if got := b.GetBatchCount(); got != 1 {
		t.Fatalf("batch count: got %d, want 1 (second call must not mint another transition)", got)
	}
}

// TestOpenBalance_ChainsAcrossTwoNewAccounts is the scenario that broke batch
// submission live: a second account appearing between two already-sealed batches used
// to silently advance the tree root with no transition recorded, so the next batch's
// OldRoot would never match the previous batch's NewRoot. With the fix, each new
// account produces its own sealed batch, and the roots must chain correctly.
func TestOpenBalance_ChainsAcrossTwoNewAccounts(t *testing.T) {
	tree := newAccountTestTree(t)
	b := batcher.NewBatcher()

	if _, err := openBalance(tree, b, newTestPubKeyHex(t), 1); err != nil {
		t.Fatalf("credit account A: %v", err)
	}
	batchA, ok := b.GetBatch(1)
	if !ok {
		t.Fatal("expected batch #1 to be sealed")
	}

	if _, err := openBalance(tree, b, newTestPubKeyHex(t), 1); err != nil {
		t.Fatalf("credit account B: %v", err)
	}
	batchB, ok := b.GetBatch(2)
	if !ok {
		t.Fatal("expected batch #2 to be sealed")
	}

	if batchB.OldRoot != batchA.NewRoot {
		t.Fatalf("batch chain broken: batch #2 OldRoot (%s) != batch #1 NewRoot (%s) — this is exactly the live bug this fix addresses", batchB.OldRoot, batchA.NewRoot)
	}
}
