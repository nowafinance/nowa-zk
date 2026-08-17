package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/spf13/cobra"
	"github.com/nowafinance/nowa-zk/prover/circuits"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Generate proving/verifying keys and Solidity verifier contract",
	Long:  `Compiles the StateTransitionCircuit, generates Groth16 keys, and exports the Solidity verifier.`,
	Run:   setup,
}

var (
	outputDir      string
	contractOutput string
)

func init() {
	// Keys output directory (default: ./keys)
	setupCmd.Flags().StringVarP(&outputDir, "output-dir", "o", "./keys", "")

	// Solidity contract output directory (default: ../contracts/src/generated)
	setupCmd.Flags().StringVarP(&contractOutput, "contract-output", "c", "../contracts/src/generated", "")
}

func setup(cmd *cobra.Command, args []string) {
	log.Println("========================================")
	log.Println("  ZK Phase 3 Unified State Circuit Setup")
	log.Println("========================================")
	log.Printf("Batch size: %d operations per proof\n", circuits.BatchSize)
	log.Println()

	// Step 1: Compile circuit
	log.Println("📦 Compiling circuit...")
	var circuit circuits.StateTransitionCircuit
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	if err != nil {
		log.Fatalf("❌ Failed to compile circuit: %v", err)
	}
	log.Printf("✅ Circuit compiled (constraints: %d)\n", ccs.GetNbConstraints())
	log.Println()

	// Step 2: Run Groth16 setup
	log.Println("🔑 Running Groth16 trusted setup...")
	log.Println("   (This may take 30-60 seconds...)")
	pk, vk, err := groth16.Setup(ccs)
	if err != nil {
		log.Fatalf("❌ Failed to run setup: %v", err)
	}
	log.Println("✅ Trusted setup completed")
	log.Println()

	// Step 3: Create output directories
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("❌ Failed to create output directory: %v", err)
	}
	if err := os.MkdirAll(contractOutput, 0755); err != nil {
		log.Fatalf("❌ Failed to create contract output directory: %v", err)
	}

	// Step 4: Save proving key
	log.Println("💾 Saving proving key...")
	pkPath := filepath.Join(outputDir, "state.pk")
	pkFile, err := os.Create(pkPath)
	if err != nil {
		log.Fatalf("❌ Failed to create proving key file: %v", err)
	}
	defer pkFile.Close()

	pkSize, err := pk.WriteTo(pkFile)
	if err != nil {
		log.Fatalf("❌ Failed to write proving key: %v", err)
	}
	log.Printf("✅ Proving key saved: %s (%d bytes)\n", pkPath, pkSize)

	// Step 5: Save verifying key
	log.Println("💾 Saving verifying key...")
	vkPath := filepath.Join(outputDir, "state.vk")
	vkFile, err := os.Create(vkPath)
	if err != nil {
		log.Fatalf("❌ Failed to create verifying key file: %v", err)
	}
	defer vkFile.Close()

	vkSize, err := vk.WriteTo(vkFile)
	if err != nil {
		log.Fatalf("❌ Failed to write verifying key: %v", err)
	}
	log.Printf("✅ Verifying key saved: %s (%d bytes)\n", vkPath, vkSize)

	// Step 6: Save compiled circuit
	log.Println("💾 Saving compiled circuit...")
	ccsPath := filepath.Join(outputDir, "state.ccs")
	ccsFile, err := os.Create(ccsPath)
	if err != nil {
		log.Fatalf("❌ Failed to create compiled circuit file: %v", err)
	}
	defer ccsFile.Close()

	ccsSize, err := ccs.WriteTo(ccsFile)
	if err != nil {
		log.Fatalf("❌ Failed to write compiled circuit: %v", err)
	}
	log.Printf("✅ Compiled circuit saved: %s (%d bytes)\n", ccsPath, ccsSize)
	log.Println()

	// Step 7: Export Solidity verifier contract
	// Contracts import generated/Verifier.sol (contract name inside is also Verifier).
	log.Println("📝 Generating Solidity verifier contract...")
	solidityPath := filepath.Join(contractOutput, "Verifier.sol")
	solidityFile, err := os.Create(solidityPath)
	if err != nil {
		log.Fatalf("❌ Failed to create Solidity file: %v", err)
	}

	if err := vk.ExportSolidity(solidityFile); err != nil {
		solidityFile.Close()
		log.Fatalf("❌ Failed to export Solidity verifier: %v", err)
	}
	solidityFile.Close()
	log.Printf("✅ Solidity verifier saved: %s\n", solidityPath)

	// Keep StateVerifier.sol as a copy for older docs/scripts.
	legacyPath := filepath.Join(contractOutput, "StateVerifier.sol")
	in, err := os.ReadFile(solidityPath)
	if err != nil {
		log.Fatalf("❌ Failed to read Verifier.sol for legacy copy: %v", err)
	}
	if err := os.WriteFile(legacyPath, in, 0644); err != nil {
		log.Fatalf("❌ Failed to write StateVerifier.sol: %v", err)
	}
	log.Printf("✅ Legacy copy saved: %s\n", legacyPath)
	log.Println()

	// Summary
	log.Println("========================================")
	log.Println("  ✅ Setup Complete!")
	log.Println("========================================")
	log.Println("Generated files:")
	log.Printf("  📁 Keys directory: %s\n", outputDir)
	log.Println("     - state.pk (proving key)")
	log.Println("     - state.vk (verifying key)")
	log.Println("     - state.ccs (compiled circuit)")
	log.Println()
	log.Printf("  📁 Contract directory: %s\n", contractOutput)
	log.Println("     - Verifier.sol (Solidity verifier)")
	log.Println("     - StateVerifier.sol (legacy alias)")
	log.Println()
	log.Println("Next steps:")
	log.Println("  1. Deploy Verifier.sol + NowaRollup.sol to your chain")
	log.Println("  2. Start prover with: ./prover-bin start")
	log.Println("========================================")
}
