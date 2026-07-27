package main

import (
	"fmt"
	"reflect"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/std/math/emulated"
)

type TestCircuit struct {
	A emulated.Element[emulated.Secp256k1Fr] `gnark:",public"`
}
func (c *TestCircuit) Define(api frontend.API) error {
	return nil
}

func main() {
	var c TestCircuit
	ccs, _ := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &c)
	_, _, _ = groth16.Setup(ccs)
	fmt.Printf("Compiled!\n")
}
