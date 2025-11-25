package main

import (
	"log"
	"os"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/spf13/cobra"
	"github.com/tannetwork/zk-sequencer/prover/circuits"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Generate proving and verifying keys for the circuit",
	Run:   setup,
}

var (
	outputDir string
)

func init() {
	setupCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "./keys", "Directory to save the keys")
}

func setup(cmd *cobra.Command, args []string) {
	log.Println("Setting up the circuit...")

	var circuit circuits.TransferCircuit

	// compile the circuit
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	if err != nil {
		log.Fatalf("Failed to compile circuit: %v", err)
	}

	// run setup
	pk, vk, err := groth16.Setup(ccs)
	if err != nil {
		log.Fatalf("Failed to run setup: %v", err)
	}

	// create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// save proving key
	pkFile, err := os.Create(outputDir + "/transfer.pk")
	if err != nil {
		log.Fatalf("Failed to create proving key file: %v", err)
	}
	defer pkFile.Close()
	if _, err := pk.WriteTo(pkFile); err != nil {
		log.Fatalf("Failed to write proving key: %v", err)
	}

	// save verifying key
	vkFile, err := os.Create(outputDir + "/transfer.vk")
	if err != nil {
		log.Fatalf("Failed to create verifying key file: %v", err)
	}
	defer vkFile.Close()
	if _, err := vk.WriteTo(vkFile); err != nil {
		log.Fatalf("Failed to write verifying key: %v", err)
	}

	// save compiled circuit
	ccsFile, err := os.Create(outputDir + "/transfer.r1cs")
	if err != nil {
		log.Fatalf("Failed to create compiled circuit file: %v", err)
	}
	defer ccsFile.Close()
	if _, err := ccs.WriteTo(ccsFile); err != nil {
		log.Fatalf("Failed to write compiled circuit: %v", err)
	}

	log.Printf("Successfully generated keys and compiled circuit in %s", outputDir)
}