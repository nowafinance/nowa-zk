package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/math/emulated"
	"github.com/nowafinance/nowa-zk/prover/circuits"
)

func main() {
	var circuit circuits.BatchTradeSignatureCircuit
	circuit.BatchRoot = 12345 // some known value
	for i := 0; i < circuits.TradeBatchSize; i++ {
		circuit.MessageHashes[i] = emulated.ValueOf[emulated.Secp256k1Fr](1)
		circuit.PubKeys[i].X = emulated.ValueOf[emulated.Secp256k1Fp](1)
		circuit.PubKeys[i].Y = emulated.ValueOf[emulated.Secp256k1Fp](1)
		circuit.Sigs[i].R = emulated.ValueOf[emulated.Secp256k1Fr](1)
		circuit.Sigs[i].S = emulated.ValueOf[emulated.Secp256k1Fr](1)
	}
	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	if err != nil {
		panic(err)
	}
	pubWitness, err := witness.Public()
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	pubWitness.WriteTo(&buf)
	data := buf.Bytes()
	nbPublic := binary.BigEndian.Uint32(data[0:4])
	fmt.Printf("nbPublic: %d\n", nbPublic)
	offset := 12
	for i := 0; i < 121; i++ {
		if offset+32 > len(data) {
			break
		}
		val := new(big.Int).SetBytes(data[offset : offset+32])
		if i < 5 {
			fmt.Printf("input[%d]: %s\n", i, val.String())
		}
		offset += 32
	}
}
