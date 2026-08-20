package main

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
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

// TestGetOrCreateAccountForDeposit_NoPhantomCredit is a regression test for a second
// bug found alongside the first: new accounts created via the deposit path used to
// get seeded with the same 1,000,000 "lab credit" used for trading-onboarding, so a
// deposit of 500 into a brand-new account ended up as a balance of 1,000,500.
func TestGetOrCreateAccountForDeposit_NoPhantomCredit(t *testing.T) {
	tree := newTestTree(t) // shared helper from apply_trade_test.go — handles tempdir + cleanup

	acc, err := getOrCreateAccountForDeposit(tree, "0xabc123", big.NewInt(1), big.NewInt(2), 1)
	if err != nil {
		t.Fatal(err)
	}
	if acc.Balance.Sign() != 0 {
		t.Fatalf("expected a freshly-created deposit account to start at balance 0, got %s", acc.Balance.String())
	}
}
