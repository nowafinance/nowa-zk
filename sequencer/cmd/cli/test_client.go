package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
)

type OrderRequest struct {
	MakerAddress string `json:"maker_address"`
	TokenID      uint32 `json:"token_id"`
	Amount       string `json:"amount"` 
	Price        string `json:"price"`  
	IsBuy        bool   `json:"is_buy"`
	Nonce        uint64 `json:"nonce"`
	Signature    string `json:"signature"`
}

func main() {
	fmt.Println("--- Generating Dummy Users (Alice & Bob) ---")
	
	// 1. Generate EdDSA Keys for Alice
	alicePriv, _ := eddsa.GenerateKey(rand.Reader)
	alicePubBytes := alicePriv.PublicKey.Bytes()
	alicePubKeyHex := "0x" + hex.EncodeToString(alicePubBytes)
	
	// 2. Generate EdDSA Keys for Bob
	bobPriv, _ := eddsa.GenerateKey(rand.Reader)
	bobPubBytes := bobPriv.PublicKey.Bytes()
	bobPubKeyHex := "0x" + hex.EncodeToString(bobPubBytes)

	fmt.Printf("Alice PubKey: %s\n", alicePubKeyHex)
	fmt.Printf("Bob PubKey:   %s\n\n", bobPubKeyHex)

	fmt.Println("--- Alice submits a BUY order for 100 tokens at price 50 ---")
	
	// Create Alice's BUY order
	aliceOrder := OrderRequest{
		MakerAddress: alicePubKeyHex,
		TokenID:      1,
		Amount:       "100",
		Price:        "50",
		IsBuy:        true,
		Nonce:        1,
	}

	// Hash Alice's order using SHA256 (same as frontend will do)
	msgStrAlice := fmt.Sprintf("%s:%d:%s:%s:%t:%d", aliceOrder.MakerAddress, aliceOrder.TokenID, aliceOrder.Amount, aliceOrder.Price, aliceOrder.IsBuy, aliceOrder.Nonce)
	msgHashAlice := sha256.Sum256([]byte(msgStrAlice))
	msgHashAlice[0] = 0 // ensure < modulus
	
	// Sign with MiMC hash as internal EdDSA logic
	hFunc := mimc.NewMiMC()
	aliceSig, err := alicePriv.Sign(msgHashAlice[:], hFunc)
	if err != nil {
		fmt.Println("Alice Sign Error:", err)
	}
	aliceOrder.Signature = "0x" + hex.EncodeToString(aliceSig)

	// POST Alice's order
	postOrder(aliceOrder)


	time.Sleep(1 * time.Second)


	fmt.Println("\n--- Bob submits a SELL order for 50 tokens at price 50 ---")
	
	bobOrder := OrderRequest{
		MakerAddress: bobPubKeyHex,
		TokenID:      1,
		Amount:       "50",
		Price:        "50",
		IsBuy:        false,
		Nonce:        1,
	}

	msgStrBob := fmt.Sprintf("%s:%d:%s:%s:%t:%d", bobOrder.MakerAddress, bobOrder.TokenID, bobOrder.Amount, bobOrder.Price, bobOrder.IsBuy, bobOrder.Nonce)
	msgHashBob := sha256.Sum256([]byte(msgStrBob))
	msgHashBob[0] = 0 // ensure < modulus
	
	bobSig, err := bobPriv.Sign(msgHashBob[:], hFunc)
	if err != nil {
		fmt.Println("Bob Sign Error:", err)
	}
	bobOrder.Signature = "0x" + hex.EncodeToString(bobSig)

	// POST Bob's order
	postOrder(bobOrder)
}

func postOrder(order OrderRequest) {
	jsonData, _ := json.Marshal(order)
	fmt.Printf("Sending Payload: %s\n", string(jsonData))
	
	resp, err := http.Post("http://localhost:8080/order", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Failed to connect to Sequencer. Is it running? Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Sequencer Response Status: %s\n", resp.Status)
	fmt.Printf("Sequencer Error: %s\n", string(body))
}
