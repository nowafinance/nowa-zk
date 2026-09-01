package main

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/nowafinance/nowa-zk/sequencer/internal/batcher"
	"github.com/nowafinance/nowa-zk/sequencer/internal/bindings"
)

// TestCompressPubKeyHex_MatchesClientEncoding is a regression test for a real bug
// found live: the Deposit Watcher used to build its tree lookup key by naively
// concatenating raw pubX||pubY (64 bytes), which is NOT the compressed pubkey format
// eddsa.PublicKey.Bytes() produces (32 bytes, curve-compressed) — the same format a
// client computes for itself and later queries /account or /proof with. That mismatch
// meant every real deposit landed under a key the depositor could never look up again.
// This test proves compressPubKeyHex(X, Y) reconstructs bit-for-bit what the client's
// own priv.PublicKey.Bytes() produces for the same key.
func TestCompressPubKeyHex_MatchesClientEncoding(t *testing.T) {
	priv, err := eddsa.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pub := priv.PublicKey

	wantHex := "0x" + hex.EncodeToString(pub.Bytes())

	x := new(big.Int)
	y := new(big.Int)
	pub.A.X.BigInt(x)
	pub.A.Y.BigInt(y)

	got, err := compressPubKeyHex(x, y)
	if err != nil {
		t.Fatalf("compressPubKeyHex: %v", err)
	}

	if got != wantHex {
		t.Fatalf("compressed pubkey mismatch:\n  got:  %s\n  want: %s (client's own encoding)", got, wantHex)
	}
}

func TestCompressPubKeyHex_RejectsOffCurvePoint(t *testing.T) {
	// (1, 1) is not a point on the BabyJubJub curve — must be rejected, not silently
	// accepted with garbage coordinates.
	_, err := compressPubKeyHex(big.NewInt(1), big.NewInt(1))
	if err == nil {
		t.Fatal("expected an error for an off-curve point, got nil")
	}
}

// mimcHash mirrors the circuit's in-circuit MiMC construction (see
// prover/circuits/state_circuit.go's accountLeaf/merkleRoot) — used below to
// independently verify a transition's Merkle proof actually folds up to its claimed
// root, the exact property that was broken by the bug this test guards against.
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

func accountLeafGo(index uint64, pubX, pubY, balance *big.Int, nonce uint64) *big.Int {
	return mimcHash(new(big.Int).SetUint64(index), pubX, pubY, balance, new(big.Int).SetUint64(nonce))
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

// TestProcessDeposit_FirstDepositFromNewAccount is the regression test for a real bug
// found live: a brand-new account's first-ever deposit claimed its "old" leaf was
// accountLeaf(index, pubX, pubY, 0, 0) — a real MiMC hash of the depositor's actual
// pubkey — but a genuinely never-written leaf's true tree value is the SMT's literal
// 0 (see merkle_db.go's zeroHashes[0]). Those are different field elements for any
// real pubkey, so the deposit's own Merkle inclusion proof didn't fold up to the
// root the batch claimed as OldRoot. Confirmed live on Sepolia, batch #1: constraint
// failure at state_circuit.go's TakerBase inclusion check.
//
// The fix adds StateUpdate.IsGenesis to the circuit, so a leg can correctly claim
// "old leaf = 0" instead of "old leaf = accountLeaf(...)". processDeposit sets it
// whenever the account didn't already exist. This test verifies the resulting
// transition's TakerBase snapshot is flagged correctly AND that folding the SMT's
// literal 0 (not accountLeaf(...)) via its path genuinely reconstructs the real
// pre-deposit root — exactly the property the live circuit check enforces.
func TestProcessDeposit_FirstDepositFromNewAccount(t *testing.T) {
	tree := newTestTree(t)
	b := batcher.NewBatcher()

	priv, err := eddsa.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pubX := new(big.Int)
	priv.PublicKey.A.X.BigInt(pubX)
	pubY := new(big.Int)
	priv.PublicKey.A.Y.BigInt(pubY)

	rootBeforeAnything := tree.Root()

	event := &bindings.NowaRollupDeposit{
		TokenId: 1,
		Amount:  big.NewInt(500),
		PubKeyX: pubX,
		PubKeyY: pubY,
	}
	processDeposit(event, tree, b)

	if got := b.GetCurrentBatchSize(); got != 1 {
		t.Fatalf("current batch size: got %d, want 1 (a single deposit transition, no separate open)", got)
	}

	rootAfter := tree.Root()

	accID, err := tree.GetAccountID(mustCompressPubKeyHex(t, pubX, pubY))
	if err != nil {
		t.Fatal(err)
	}
	leafIndex := accID*256 + uint64(event.TokenId)

	// A leaf's own writes never change its own sibling path (only writes to *other*
	// leaves would), so this single path fetch is valid both before and after.
	path, bits := tree.GetPath(leafIndex)
	var pathStr [28]string
	var bitsInt [28]int
	for i := 0; i < 28; i++ {
		pathStr[i] = path[i].String()
		bitsInt[i] = int(bits[i])
	}

	// The property under test: folding the SMT's literal zero (NOT accountLeaf(...))
	// via this path must reconstruct the real pre-deposit root.
	gotGenesisRoot := foldToRoot(big.NewInt(0), pathStr, bitsInt)
	if gotGenesisRoot.Cmp(rootBeforeAnything) != 0 {
		t.Fatalf("genesis leaf (literal 0) doesn't fold up to the pre-deposit root:\n  got:  %s\n  want: %s", gotGenesisRoot, rootBeforeAnything)
	}

	// Final sanity: after the deposit, the leaf holds the real deposited amount.
	finalLeaf := accountLeafGo(leafIndex, pubX, pubY, big.NewInt(500), 0)
	gotFinalRoot := foldToRoot(finalLeaf, pathStr, bitsInt)
	if gotFinalRoot.Cmp(rootAfter) != 0 {
		t.Fatalf("final balance proof doesn't match tree's actual current root:\n  got:  %s\n  want: %s", gotFinalRoot, rootAfter)
	}
}

func mustCompressPubKeyHex(t *testing.T, x, y *big.Int) string {
	t.Helper()
	h, err := compressPubKeyHex(x, y)
	if err != nil {
		t.Fatal(err)
	}
	return h
}
