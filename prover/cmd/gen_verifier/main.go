package main

import (
	"log"
	"os"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/nowafinance/nowa-zk/prover/circuits"
)

func main() {
	log.Println("Compiling StateTransitionCircuit...")
	var circuit circuits.StateTransitionCircuit
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	if err != nil {
		log.Fatalf("Compilation failed: %v", err)
	}

	log.Println("Running Groth16 Setup...")
	pk, vk, err := groth16.Setup(ccs)
	if err != nil {
		log.Fatalf("Setup failed: %v", err)
	}
	_ = pk

	log.Println("Exporting Verifier.sol...")
	err = os.MkdirAll("../../contracts/src/generated", 0755)
	if err != nil {
		log.Fatalf("Mkdir failed: %v", err)
	}
	f, err := os.Create("../../contracts/src/generated/Verifier.sol")
	if err != nil {
		log.Fatalf("Create failed: %v", err)
	}
	defer f.Close()

	err = vk.ExportSolidity(f)
	if err != nil {
		log.Fatalf("ExportSolidity failed: %v", err)
	}
	log.Println("✅ Verifier.sol generated successfully.")
}
