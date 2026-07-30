package main

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/consensys/gnark-crypto/ecc/twistededwards"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	gnark_eddsa "github.com/consensys/gnark/std/signature/eddsa"
	gnark_mimc "github.com/consensys/gnark/std/hash/mimc"
	std_twistededwards "github.com/consensys/gnark/std/algebra/native/twistededwards"
	"github.com/consensys/gnark/backend/groth16"
)

type eddsaCircuit struct {
	PublicKey gnark_eddsa.PublicKey   `gnark:",public"`
	Signature gnark_eddsa.Signature   `gnark:",public"`
	Message   frontend.Variable       `gnark:",public"`
}

func (circuit *eddsaCircuit) Define(api frontend.API) error {
	curve, err := std_twistededwards.NewEdCurve(api, twistededwards.BN254)
	if err != nil {
		return err
	}
	mimcHash, err := gnark_mimc.NewMiMC(api)
	if err != nil {
		return err
	}
	return gnark_eddsa.Verify(curve, circuit.Signature, circuit.Message, circuit.PublicKey, &mimcHash)
}

func main() {
	privKey, _ := eddsa.GenerateKey(rand.Reader)
	pubKey := privKey.PublicKey

	msgInt := new(big.Int).SetInt64(42)
	msgData := make([]byte, 32)
	msgInt.FillBytes(msgData)

	hFunc := mimc.NewMiMC()
	sigBin, _ := privKey.Sign(msgData, hFunc)

	var witness eddsaCircuit
	witness.Message = msgInt
	witness.PublicKey.Assign(twistededwards.BN254, pubKey.Bytes())
	witness.Signature.Assign(twistededwards.BN254, sigBin)

	fmt.Println("Compiling circuit...")
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &eddsaCircuit{})
	if err != nil {
		fmt.Println("Compile err:", err)
		return
	}

	fmt.Println("Creating witness...")
	witnessFull, err := frontend.NewWitness(&witness, ecc.BN254.ScalarField())
	if err != nil {
		fmt.Println("Witness err:", err)
		return
	}

	fmt.Println("IsSolved...")
	err = ccs.IsSolved(witnessFull)
	fmt.Println("IsSolved err:", err)
	
	// Just do a dummy setup and prove
	pk, _, err := groth16.Setup(ccs)
	if err != nil {
		fmt.Println("Setup err:", err)
		return
	}
	_, err = groth16.Prove(ccs, pk, witnessFull)
	fmt.Println("Prove err:", err)
}
