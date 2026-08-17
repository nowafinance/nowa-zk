package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
)

type OrderRequest struct {
	MakerAddress     string `json:"maker_address"`
	TokenID          uint32 `json:"token_id"`
	Amount           string `json:"amount"`
	Price            string `json:"price"`
	IsBuy            bool   `json:"is_buy"`
	Nonce            uint64 `json:"nonce"`
	Signature        string `json:"signature"`
	CircuitSignature string `json:"circuit_signature"`
}

// accountInfo/accountResp mirror the shape returned by GET /account.
type accountInfo struct {
	Index   uint64 `json:"index"`
	PubKeyX string `json:"pub_key_x"`
	PubKeyY string `json:"pub_key_y"`
}
type accountResp struct {
	Base  accountInfo `json:"base"`
	Quote accountInfo `json:"quote"`
}

var sequencerURL string

func main() {
	count := flag.Int("count", 1, "number of matched trades to generate in this run")
	force := flag.Bool("force", false, "run even if a lock file from a previous run exists")
	flag.StringVar(&sequencerURL, "sequencer-url", "http://localhost:8080", "Sequencer REST API base URL")
	flag.Parse()

	// --- One-shot guard ---
	//
	// GET /account lab-credits a brand-new pubkey by writing directly to the Merkle
	// tree (sequencer/internal/api/account.go's openBalance) WITHOUT recording a
	// StateTransition. That's invisible to the batcher/prover. Every run of this tool
	// mints a fresh random Alice/Bob keypair, so a *second* run — even asking for a
	// different --count — silently advances the tree root in between whatever batches
	// were already sealed, breaking the chain the Prover needs (batch N's old_root
	// must equal batch N-1's new_root). We hit this live: batch #1 settled fine, a
	// second run's batch #2 could never be submitted, no recovery possible short of
	// redeploying the contract. So: refuse to run twice unless forced.
	lockPath := lockFilePath()
	if !*force {
		if data, err := os.ReadFile(lockPath); err == nil {
			fmt.Printf("⛔ Refusing to run — lock file already exists: %s\n", lockPath)
			fmt.Printf("   (%s)\n", string(data))
			fmt.Println("   Running again mints a NEW random keypair and will silently break the")
			fmt.Println("   Sequencer's batch chain — see docs/operations/troubleshooting.md.")
			fmt.Println("   Safe options:")
			fmt.Println("     - Only run this once per Sequencer/contract lifetime.")
			fmt.Println("     - After a fresh `make deploy` + `make clean-sequencer-state` (+ Sequencer")
			fmt.Println("       restart), delete the lock file to run again from a clean slate.")
			fmt.Println("     - Pass --force if you understand the risk (e.g. you don't care about")
			fmt.Println("       submitting further batches to L1, just exercising the matching engine).")
			os.Exit(1)
		}
	}

	fmt.Printf("--- Generating %d matched trade(s) between one Alice/Bob pair ---\n", *count)

	alicePriv, _ := eddsa.GenerateKey(rand.Reader)
	alicePubKeyHex := "0x" + hex.EncodeToString(alicePriv.PublicKey.Bytes())
	bobPriv, _ := eddsa.GenerateKey(rand.Reader)
	bobPubKeyHex := "0x" + hex.EncodeToString(bobPriv.PublicKey.Bytes())

	fmt.Printf("Alice PubKey: %s\n", alicePubKeyHex)
	fmt.Printf("Bob PubKey:   %s\n\n", bobPubKeyHex)

	// Register both accounts EXACTLY ONCE, before any trade — this is the only place
	// in the whole run that can trigger the untracked mutation, and it happens before
	// any batch involving them is sealed, so it can't land *between* two sealed batches.
	fmt.Println("--- Registering accounts (GET /account, once) ---")
	aliceAcc := fetchAccount(alicePubKeyHex)
	bobAcc := fetchAccount(bobPubKeyHex)
	fmt.Printf("Alice: base_idx=%d quote_idx=%d\n", aliceAcc.Base.Index, aliceAcc.Quote.Index)
	fmt.Printf("Bob:   base_idx=%d quote_idx=%d\n\n", bobAcc.Base.Index, bobAcc.Quote.Index)

	// Circuit signatures only cover (OpType, pubX, pubY, baseIndex, quoteIndex) — none
	// of that changes between rounds for a fixed pair of already-registered accounts,
	// so sign once and reuse for every trade (see state_circuit.go's processOperation:
	// the fill amount is deliberately not part of this message either).
	aliceCircuitSig := circuitSignature(alicePriv, aliceAcc.Base.PubKeyX, aliceAcc.Base.PubKeyY, aliceAcc.Base.Index, aliceAcc.Quote.Index)
	bobCircuitSig := circuitSignature(bobPriv, bobAcc.Base.PubKeyX, bobAcc.Base.PubKeyY, bobAcc.Base.Index, bobAcc.Quote.Index)

	startCount, _ := fetchBatchCount()

	failures := 0
	for i := 0; i < *count; i++ {
		nonce := uint64(i)

		// Bob rests first as the SELL maker; Alice's BUY matches him as taker.
		bobOrder := OrderRequest{MakerAddress: bobPubKeyHex, TokenID: 1, Amount: "10", Price: "50", IsBuy: false, Nonce: nonce}
		signOrderIntent(&bobOrder, bobPriv)
		bobOrder.CircuitSignature = bobCircuitSig
		if !postOrderQuiet(bobOrder) {
			failures++
		}

		aliceOrder := OrderRequest{MakerAddress: alicePubKeyHex, TokenID: 1, Amount: "10", Price: "50", IsBuy: true, Nonce: nonce}
		signOrderIntent(&aliceOrder, alicePriv)
		aliceOrder.CircuitSignature = aliceCircuitSig
		if !postOrderQuiet(aliceOrder) {
			failures++
		}

		if (i+1)%50 == 0 || i+1 == *count {
			fmt.Printf("  ... %d/%d round(s) placed\n", i+1, *count)
		}
	}

	time.Sleep(500 * time.Millisecond) // let the matching goroutine finish sealing the last batch
	endCount, err := fetchBatchCount()
	if err != nil {
		fmt.Println("Warning: could not confirm final batch count:", err)
	}
	delta := int64(endCount) - int64(startCount)
	fmt.Printf("\n✅ Done. Sequencer batch count: %d → %d (Δ=%d, requested=%d, order failures=%d)\n",
		startCount, endCount, delta, *count, failures)
	if delta != int64(*count) {
		fmt.Println("⚠️  Sealed batch count didn't increase by exactly --count — check /orderbook and the Sequencer's log.")
	}

	if failures == 0 {
		if err := os.MkdirAll(filepath.Dir(lockPath), 0755); err != nil {
			fmt.Println("Warning: could not create lock directory:", err)
			return
		}
		msg := fmt.Sprintf("ran at %s, count=%d, alice=%s, bob=%s\n", time.Now().Format(time.RFC3339), *count, alicePubKeyHex, bobPubKeyHex)
		if err := os.WriteFile(lockPath, []byte(msg), 0644); err != nil {
			fmt.Println("Warning: could not write lock file:", err)
			return
		}
		fmt.Printf("🔒 Lock file written: %s\n", lockPath)
	} else {
		fmt.Println("⚠️  Not writing the lock file since some orders failed — safe to retry.")
	}
}

func lockFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".nowa-zk", "sequencer", "test_client.lock")
}

// hashGo is the native-side equivalent of the circuit's in-circuit MiMC hashing
// (see prover/circuits/state_circuit.go and state_circuit_test.go).
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

func toBig(s string) *big.Int {
	n := new(big.Int)
	n.SetString(s, 10)
	return n
}

// signOrderIntent signs the SHA256 order payload — this is what the Sequencer's
// matching engine checks (engine.VerifyEdDSASignature) to confirm trade intent.
func signOrderIntent(o *OrderRequest, priv *eddsa.PrivateKey) {
	msgStr := fmt.Sprintf("%s:%d:%s:%s:%t:%d", o.MakerAddress, o.TokenID, o.Amount, o.Price, o.IsBuy, o.Nonce)
	msgHash := sha256.Sum256([]byte(msgStr))
	msgHash[0] = 0 // ensure < modulus
	sig, err := priv.Sign(msgHash[:], mimc.NewMiMC())
	if err != nil {
		fmt.Println("Sign Error:", err)
	}
	o.Signature = "0x" + hex.EncodeToString(sig)
}

// circuitSignature signs the exact message the StateTransitionCircuit verifies for
// a Trade op: MiMC(OpType=0, pubX, pubY, baseIndex, quoteIndex). This authorizes the
// resting order for any partial fill under one signature (the fill amount itself is
// intentionally not part of the message — see state_circuit.go's processOperation).
func circuitSignature(priv *eddsa.PrivateKey, pubX, pubY string, baseIndex, quoteIndex uint64) string {
	msgBig := hashGo(big.NewInt(0), toBig(pubX), toBig(pubY), new(big.Int).SetUint64(baseIndex), new(big.Int).SetUint64(quoteIndex))
	var msgFr fr.Element
	msgFr.SetBigInt(msgBig)
	msgBytes := msgFr.Bytes()
	sig, err := priv.Sign(msgBytes[:], mimc.NewMiMC())
	if err != nil {
		fmt.Println("Circuit Sign Error:", err)
	}
	return "0x" + hex.EncodeToString(sig)
}

func fetchAccount(pubKeyHex string) accountResp {
	resp, err := http.Get(fmt.Sprintf("%s/account?pubkey=%s", sequencerURL, pubKeyHex))
	if err != nil {
		fmt.Printf("Failed to fetch account. Is the Sequencer running? Error: %v\n", err)
		return accountResp{}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out accountResp
	if err := json.Unmarshal(body, &out); err != nil {
		fmt.Printf("Failed to decode /account response: %v (body: %s)\n", err, body)
	}
	return out
}

func fetchBatchCount() (uint64, error) {
	resp, err := http.Get(sequencerURL + "/batch/count")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var out struct {
		Count uint64 `json:"count"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &out); err != nil {
		return 0, fmt.Errorf("decode /batch/count: %w (body: %s)", err, body)
	}
	return out.Count, nil
}

// postOrderQuiet is like postOrder but doesn't print the full payload every round —
// only failures, so a --count 1000 run doesn't flood the terminal. Returns true on 200.
func postOrderQuiet(order OrderRequest) bool {
	jsonData, _ := json.Marshal(order)
	resp, err := http.Post(sequencerURL+"/order", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to connect to Sequencer. Is it running? Error: %v\n", err)
		return false
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Order rejected (%s): %s\nPayload: %s\n", resp.Status, string(body), string(jsonData))
		return false
	}
	return true
}
