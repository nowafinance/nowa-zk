package api

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"os"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/nowafinance/nowa-zk/sequencer/internal/batcher"
	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
)

// mimcHash/foldToRoot mirror the circuit's in-circuit MiMC construction (see
// prover/circuits/state_circuit.go's accountLeaf/merkleRoot) — used below to
// independently verify a transition's Merkle proof actually folds up to its claimed
// root, the same property that was broken by the deposit-genesis bug this guards
// against (sequencer/cmd/sequencer/deposit_watcher_test.go has the full history).
func mimcHash(items ...*big.Int) *big.Int {
	h := mimc.NewMiMC()
	for _, it := range items {
		var f fr.Element
		f.SetBigInt(it)
		b := f.Bytes()
		h.Write(b[:])
	}
	res := new(big.Int)
	res.SetBytes(h.Sum(nil))
	return res
}

func foldToRoot(leaf *big.Int, path [28]string, bits [28]int) *big.Int {
	cur := leaf
	for i := 0; i < 28; i++ {
		sib, _ := new(big.Int).SetString(path[i], 10)
		if bits[i] == 1 {
			cur = mimcHash(sib, cur)
		} else {
			cur = mimcHash(cur, sib)
		}
	}
	return cur
}

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
// asserts the credit now shows up as a real, batcher-tracked OpDeposit transition.
//
// A batch only seals once BatchSize real transitions have accumulated (see
// batcher.BatchSize), so this credits BatchSize-1 filler accounts first (each
// tracked, none sealing anything yet), then the account under test as the exact
// transition that completes and seals the batch — letting this test inspect a real
// sealed batch rather than needing a batcher API for peeking at an open one.
func TestOpenBalance_TracksLabCreditAsTransition(t *testing.T) {
	tree := newAccountTestTree(t)
	b := batcher.NewBatcher()

	rootAtBatchStart := tree.Root().String()

	for i := 0; i < batcher.BatchSize-1; i++ {
		if _, err := openBalance(tree, b, newTestPubKeyHex(t), 1); err != nil {
			t.Fatalf("filler account %d: %v", i, err)
		}
	}
	if b.GetLatestBatch() != nil {
		t.Fatal("batch should not seal before the BatchSize-th transition")
	}
	rootBeforeTestAccount := tree.Root()

	acc, err := openBalance(tree, b, newTestPubKeyHex(t), 1)
	if err != nil {
		t.Fatalf("openBalance: %v", err)
	}
	if acc.Balance.String() != "1000000" {
		t.Fatalf("balance: got %s, want 1000000", acc.Balance)
	}

	rootAfter := tree.Root().String()

	sealed := b.GetLatestBatch()
	if sealed == nil {
		t.Fatal("expected a sealed batch after the BatchSize-th lab-credit transition, got none")
	}
	if len(sealed.Transitions) != batcher.BatchSize {
		t.Fatalf("expected exactly %d transitions, got %d", batcher.BatchSize, len(sealed.Transitions))
	}
	// The account under test was the LAST credit in this batch.
	tr := sealed.Transitions[batcher.BatchSize-1]
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
	if !tr.TakerBase.IsGenesis {
		t.Fatal("TakerBase.IsGenesis: got false, want true (this leaf was never written before)")
	}
	// The actual circuit-level property: folding the SMT's literal zero (not
	// accountLeaf(...)) via this leg's path must reconstruct the real root the tree
	// was at immediately before this specific credit — not just the batch's overall
	// start (24 filler credits happened first).
	var pathStr [28]string
	var bitsInt [28]int
	for i := 0; i < 28; i++ {
		pathStr[i] = tr.TakerBase.Path[i]
		bitsInt[i] = int(tr.TakerBase.PathBits[i])
	}
	gotGenesisRoot := foldToRoot(big.NewInt(0), pathStr, bitsInt)
	if gotGenesisRoot.Cmp(rootBeforeTestAccount) != 0 {
		t.Fatalf("genesis leaf (literal 0) doesn't fold up to the root before this credit:\n  got:  %s\n  want: %s", gotGenesisRoot, rootBeforeTestAccount)
	}
	if sealed.OldRoot != rootAtBatchStart {
		t.Fatalf("batch OldRoot: got %s, want %s (tree root before any credit in this batch)", sealed.OldRoot, rootAtBatchStart)
	}
	if sealed.NewRoot != rootAfter {
		t.Fatalf("batch NewRoot: got %s, want %s (tree root after all %d credits)", sealed.NewRoot, rootAfter, batcher.BatchSize)
	}
}

// TestOpenBalance_SecondCallDoesNotDuplicateCredit confirms an already-funded account
// is returned as-is, with no second transition minted — checked via the *current*
// (still-open, pre-seal) batch size, since a single account's two calls never reaches
// BatchSize on their own.
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
	if got := b.GetCurrentBatchSize(); got != 1 {
		t.Fatalf("current batch size: got %d, want 1 (second call must not mint another transition)", got)
	}
}

// TestOpenBalance_ChainsAcrossTwoNewAccounts is the scenario that broke batch
// submission live: a new account appearing used to silently advance the tree root
// with no transition recorded, so the next batch's OldRoot would never match the
// previous batch's NewRoot. With the fix, every credit is a tracked transition, and
// two full batches of new accounts must chain roots correctly across the boundary.
func TestOpenBalance_ChainsAcrossTwoNewAccounts(t *testing.T) {
	tree := newAccountTestTree(t)
	b := batcher.NewBatcher()

	for i := 0; i < batcher.BatchSize; i++ {
		if _, err := openBalance(tree, b, newTestPubKeyHex(t), 1); err != nil {
			t.Fatalf("batch #1 account %d: %v", i, err)
		}
	}
	batchA, ok := b.GetBatch(1)
	if !ok {
		t.Fatal("expected batch #1 to be sealed")
	}

	for i := 0; i < batcher.BatchSize; i++ {
		if _, err := openBalance(tree, b, newTestPubKeyHex(t), 1); err != nil {
			t.Fatalf("batch #2 account %d: %v", i, err)
		}
	}
	batchB, ok := b.GetBatch(2)
	if !ok {
		t.Fatal("expected batch #2 to be sealed")
	}

	if batchB.OldRoot != batchA.NewRoot {
		t.Fatalf("batch chain broken: batch #2 OldRoot (%s) != batch #1 NewRoot (%s) — this is exactly the live bug this fix addresses", batchB.OldRoot, batchA.NewRoot)
	}
}
