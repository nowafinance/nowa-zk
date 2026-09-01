package api

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

// hashGo mirrors the native-side MiMC hashing used throughout this codebase
// (see sequencer/cmd/cli/test_client.go, prover/circuits/state_circuit_test.go) —
// used here to independently re-derive the root from a /proof response and confirm
// it actually matches the tree's real root, not just that the JSON is well-formed.
func hashGo(items ...*big.Int) *big.Int {
	h := mimc.NewMiMC()
	for _, item := range items {
		var f fr.Element
		f.SetBigInt(item)
		b := f.Bytes()
		h.Write(b[:])
	}
	res := new(big.Int)
	res.SetBytes(h.Sum(nil))
	return res
}

func toBig(t *testing.T, s string) *big.Int {
	t.Helper()
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("not a valid decimal string: %q", s)
	}
	return n
}

// TestHandleProof_FoldsUpToRealRoot is the strongest check available for this
// endpoint: it doesn't just assert the response shape, it independently recomputes
// MiMC(index, pubX, pubY, balance, nonce) and folds it up through the returned
// siblings/path_bits using the exact same convention as the circuit
// (prover/circuits/state_circuit.go's merkleRoot/accountLeaf) and NowaRollup.sol's
// emergencyWithdraw, then asserts the result equals the tree's actual current root.
// If this passes, the proof this endpoint serves is genuinely usable on-chain.
func TestHandleProof_FoldsUpToRealRoot(t *testing.T) {
	dir, err := os.MkdirTemp("", "nowa-proof-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tree, err := state.NewLevelDBMerkleTree(dir, 28)
	if err != nil {
		t.Fatalf("open tree: %v", err)
	}
	defer tree.Close()

	// Fund an account the same way getOrCreateBalance/openBalance do — via GetAccountID
	// + SetBalance — so the fixture matches real onboarding behavior exactly.
	pubKey := "0x1234"
	accID, err := tree.GetAccountID(pubKey)
	if err != nil {
		t.Fatalf("GetAccountID: %v", err)
	}
	pubX := big.NewInt(111)
	pubY := big.NewInt(222)
	balance := big.NewInt(50000)
	if err := tree.SetBalance(&types.BalanceState{
		AccountID: accID, TokenID: 1, PubKeyX: pubX, PubKeyY: pubY, Balance: balance, Nonce: 0,
	}); err != nil {
		t.Fatalf("SetBalance: %v", err)
	}

	srv := &Server{tree: tree}
	req := httptest.NewRequest(http.MethodGet, "/proof?pubkey="+pubKey+"&token_id=1", nil)
	w := httptest.NewRecorder()
	srv.handleProof(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Index    uint64   `json:"index"`
		Balance  string   `json:"balance"`
		Nonce    uint64   `json:"nonce"`
		PubKeyX  string   `json:"pub_key_x"`
		PubKeyY  string   `json:"pub_key_y"`
		Siblings []string `json:"siblings"`
		PathBits []int    `json:"path_bits"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Siblings) != 28 || len(resp.PathBits) != 28 {
		t.Fatalf("expected 28 siblings/path_bits, got %d/%d", len(resp.Siblings), len(resp.PathBits))
	}
	if resp.Balance != "50000" {
		t.Fatalf("expected balance 50000, got %s", resp.Balance)
	}

	// Recompute the leaf exactly like accountLeaf() in state_circuit.go.
	leaf := hashGo(new(big.Int).SetUint64(resp.Index), toBig(t, resp.PubKeyX), toBig(t, resp.PubKeyY),
		toBig(t, resp.Balance), new(big.Int).SetUint64(resp.Nonce))

	// Fold up exactly like merkleRoot() in state_circuit.go / _foldMerklePath in
	// NowaRollup.sol: bit=1 means the current node is the right child.
	cur := leaf
	for i := 0; i < 28; i++ {
		sibling := toBig(t, resp.Siblings[i])
		if resp.PathBits[i] == 1 {
			cur = hashGo(sibling, cur)
		} else {
			cur = hashGo(cur, sibling)
		}
	}

	if cur.Cmp(tree.Root()) != 0 {
		t.Fatalf("recomputed root does not match tree's actual root:\n  got:  %s\n  want: %s", cur.String(), tree.Root().String())
	}
}

func TestHandleProof_RequiresPubkey(t *testing.T) {
	dir, err := os.MkdirTemp("", "nowa-proof-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	tree, err := state.NewLevelDBMerkleTree(dir, 28)
	if err != nil {
		t.Fatal(err)
	}
	defer tree.Close()

	srv := &Server{tree: tree}
	req := httptest.NewRequest(http.MethodGet, "/proof", nil)
	w := httptest.NewRecorder()
	srv.handleProof(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing pubkey, got %d", w.Code)
	}
}
