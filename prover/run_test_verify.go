package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
)

func main() {
	alicePriv, _ := eddsa.GenerateKey(rand.Reader)
	pub := alicePriv.PublicKey
	hFunc := mimc.NewMiMC()
	
	msg := make([]byte, 32)
	rand.Read(msg)
	msg[0] = 0 // stay in field
	
	sig, _ := alicePriv.Sign(msg, hFunc)
	
	// Verify
	valid, err := pub.Verify(sig, msg, hFunc)
	fmt.Println("Verify with same bytes:", valid, err)
	
	// What if msg is passed as big-endian bytes?
	msgInt := new(big.Int).SetBytes(msg)
	msgBytes2 := msgInt.Bytes()
	// Pad to 32 bytes
	pad := make([]byte, 32-len(msgBytes2))
	msgBytes2 = append(pad, msgBytes2...)
	valid2, err2 := pub.Verify(sig, msgBytes2, hFunc)
	fmt.Println("Verify with bigInt bytes:", valid2, err2)
}
