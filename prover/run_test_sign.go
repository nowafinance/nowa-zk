package main

import (
	"crypto/rand"
	"fmt"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
)

func main() {
	alicePriv, _ := eddsa.GenerateKey(rand.Reader)
	hFunc := mimc.NewMiMC()
	msg := []byte("hello")
	sig, err := alicePriv.Sign(msg, hFunc)
	fmt.Println("sig length:", len(sig), "error:", err)
}
