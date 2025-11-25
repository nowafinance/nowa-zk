package main

import (
	"log"
	"math/big"
	"os"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/spf13/cobra"
	"github.com/tannetwork/zk-sequencer/prover/circuits"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the prover",
	Run:   start,
}

var (
	keysDir string
)

func init() {
	startCmd.Flags().StringVarP(&keysDir, "keys-dir", "k", "./keys", "Directory where the keys are stored")
}

func start(cmd *cobra.Command, args []string) {
	log.Println("Starting the prover...")

	// load compiled circuit
	ccsFile, err := os.Open(keysDir + "/transfer.r1cs")
	if err != nil {
		log.Fatalf("Failed to open compiled circuit file: %v", err)
	}
	defer ccsFile.Close()
	ccs := groth16.NewCS(ecc.BN254)
	if _, err := ccs.ReadFrom(ccsFile); err != nil {
		log.Fatalf("Failed to read compiled circuit: %v", err)
	}

	// load proving key
	pkFile, err := os.Open(keysDir + "/transfer.pk")
	if err != nil {
		log.Fatalf("Failed to open proving key file: %v", err)
	}
	defer pkFile.Close()
	pk := groth16.NewProvingKey(ecc.BN254)
	if _, err := pk.ReadFrom(pkFile); err != nil {
		log.Fatalf("Failed to read proving key: %v", err)
	}

	// create a dummy witness
	witness, err := frontend.NewWitness(dummyWitness(), ecc.BN254.ScalarField())
	if err != nil {
		log.Fatalf("Failed to create witness: %v", err)
	}

	// generate proof
	proof, err := groth16.Prove(ccs, pk, witness)
	if err != nil {
		log.Fatalf("Failed to generate proof: %v", err)
	}

	log.Println("Successfully generated proof:")
	// print proof
	proof.WriteTo(os.Stdout)
}

func mimcHash(data ...*big.Int) *big.Int {
	h := mimc.NewMiMC()
	for _, d := range data {
		h.Write(d.Bytes())
	}
	return new(big.Int).SetBytes(h.Sum(nil))
}

func fromIndexAsField(index int) *big.Int {
	res := big.NewInt(0)
	pow := big.NewInt(1)
	for i := 0; i < circuits.TREE_DEPTH; i++ {
		if (index>>i)&1 == 1 {
			res.Add(res, pow)
		}
		pow.Mul(pow, big.NewInt(2))
	}
	return res
}

func dummyWitness() *circuits.TransferCircuit {
	// a consistent witness for TREE_DEPTH = 2

	// account states
	fromOldBal := big.NewInt(1000)
	fromNonce := big.NewInt(1)
	fromToken := big.NewInt(0)

	toOldBal := big.NewInt(500)
	toNonce := big.NewInt(0)
	toToken := big.NewInt(0)

	// transfer details
	amount := big.NewInt(250)
	fee := big.NewInt(5)

	// zero leaf and hashes
	h0 := mimcHash(big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0))
	h1 := mimcHash(h0, h0)

	// leaves
	l0 := mimcHash(fromIndexAsField(0), fromOldBal, fromNonce, fromToken)
	l1 := mimcHash(fromIndexAsField(1), toOldBal, toNonce, toToken)
	log.Printf("l0: %s", l0.String())
	log.Printf("l1: %s", l1.String())


	// BeforeRoot
	beforeRoot := mimcHash(mimcHash(l0, l1), h1)
	log.Printf("beforeRoot: %s", beforeRoot.String())

	// Intermediate state
	amtPlusFee := new(big.Int).Add(amount, fee)
	fromBalanceIntermediate := new(big.Int).Sub(fromOldBal, amtPlusFee)
	fromNonceIntermediate := new(big.Int).Add(fromNonce, big.NewInt(1))
	l0New := mimcHash(fromIndexAsField(0), fromBalanceIntermediate, fromNonceIntermediate, fromToken)
	intermediateRoot := mimcHash(mimcHash(l0New, l1), h1)
	log.Printf("intermediateRoot: %s", intermediateRoot.String())


	// After state
	toBalanceAfter := new(big.Int).Add(toOldBal, amount)
	l1New := mimcHash(fromIndexAsField(1), toBalanceAfter, toNonce, toToken)
	afterRoot := mimcHash(mimcHash(l0New, l1New), h1)
	log.Printf("afterRoot: %s", afterRoot.String())

	// create circuit witness
	var circuit circuits.TransferCircuit
	circuit.BeforeRoot = beforeRoot
	circuit.IntermediateRoot = intermediateRoot
	circuit.AfterRoot = afterRoot

	circuit.FromIndexBits[0] = 0
	circuit.FromIndexBits[1] = 0
	circuit.FromPathSiblings[0] = l1
	circuit.FromPathSiblings[1] = h1

	circuit.ToIndexBits[0] = 1
	circuit.ToIndexBits[1] = 0
	circuit.ToPathSiblings[0] = l0New
	circuit.ToPathSiblings[1] = h1

	circuit.FromOldBalance = fromOldBal
	circuit.FromNonce = fromNonce
	circuit.FromToken = fromToken

	circuit.ToOldBalance = toOldBal
	circuit.ToNonce = toNonce
	circuit.ToToken = toToken

	circuit.Amount = amount
	circuit.Fee = fee

	return &circuit
}
