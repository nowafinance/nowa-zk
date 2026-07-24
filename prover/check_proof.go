package main

import (
	"fmt"
	"os"

	bn254 "github.com/consensys/gnark/backend/groth16/bn254"
)

func main() {
	f, _ := os.Open(os.ExpandEnv("$HOME/.nowa-zk/prover/failures/batch_1_proof.bin"))
	proof := bn254.Proof{}
	_, err := proof.ReadFrom(f)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Commitments length: %d\n", len(proof.Commitments))
}
