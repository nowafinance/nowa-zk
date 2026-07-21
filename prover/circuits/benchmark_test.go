package circuits

import (
	"math/big"
	"testing"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// BenchmarkCircuitSetup measures the time to compile the circuit and run the trusted setup.
func BenchmarkCircuitSetup(b *testing.B) {
	emptyCircuit := Circuit{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &emptyCircuit)
		if err != nil {
			b.Fatal(err)
		}
		_, _, err = groth16.Setup(ccs)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("Setup Time: %s", time.Since(start))
	}
}

// BenchmarkWitnessGeneration measures the time to generate the witness from inputs.
func BenchmarkWitnessGeneration(b *testing.B) {
	// Prepare data outside the loop
	transactions := make([]Transaction, BatchSize)
	for i := 0; i < BatchSize; i++ {
		transactions[i] = Transaction{
			Nonce:    big.NewInt(int64(i)),
			From:     big.NewInt(int64(1000 + i)),
			To:       big.NewInt(int64(2000 + i)),
			Amount:   big.NewInt(int64(100 + i)),
			GasPrice: big.NewInt(int64(20 * 1e9)),
			GasLimit: big.NewInt(int64(21000 + i)),
			Data:     big.NewInt(int64(3000 + i)),
		}
	}

	expectedRoot := computeMerkleRootBench(transactions)
	expectedStateRoot := computeStateRootBench(big.NewInt(0), transactions)

	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = expectedRoot
	circuit.PrevStateRoot = big.NewInt(0)
	circuit.NewStateRoot = expectedStateRoot
	circuit.BatchNumber = big.NewInt(1)
	circuit.Timestamp = big.NewInt(1700000000)
	circuit.IndexerAddr = big.NewInt(9999)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("Witness Generation Time: %s", time.Since(start))
	}
}

// BenchmarkProofGeneration measures ONLY the proof generation time (Groth16 Prove).
func BenchmarkProofGeneration(b *testing.B) {
	// Setup Phase
	transactions := make([]Transaction, BatchSize)
	for i := 0; i < BatchSize; i++ {
		transactions[i] = Transaction{
			Nonce:    big.NewInt(int64(i)),
			From:     big.NewInt(int64(1000 + i)),
			To:       big.NewInt(int64(2000 + i)),
			Amount:   big.NewInt(int64(100 + i)),
			GasPrice: big.NewInt(int64(20 * 1e9)),
			GasLimit: big.NewInt(int64(21000 + i)),
			Data:     big.NewInt(int64(3000 + i)),
		}
	}

	expectedRoot := computeMerkleRootBench(transactions)
	expectedStateRoot := computeStateRootBench(big.NewInt(0), transactions)

	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = expectedRoot
	circuit.PrevStateRoot = big.NewInt(0)
	circuit.NewStateRoot = expectedStateRoot
	circuit.BatchNumber = big.NewInt(1)
	circuit.Timestamp = big.NewInt(1700000000)
	circuit.IndexerAddr = big.NewInt(9999)

	// Compile & Setup
	emptyCircuit := Circuit{}
	ccs, _ := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &emptyCircuit)
	pk, _, _ := groth16.Setup(ccs)
	witness, _ := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := groth16.Prove(ccs, pk, witness)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("Proof Generation Time: %s", time.Since(start))
	}
}

// BenchmarkVerification measures the time to verify a proof.
func BenchmarkVerification(b *testing.B) {
	// Setup Phase
	transactions := make([]Transaction, BatchSize)
	for i := 0; i < BatchSize; i++ {
		transactions[i] = Transaction{
			Nonce:    big.NewInt(int64(i)),
			From:     big.NewInt(int64(1000 + i)),
			To:       big.NewInt(int64(2000 + i)),
			Amount:   big.NewInt(int64(100 + i)),
			GasPrice: big.NewInt(int64(20 * 1e9)),
			GasLimit: big.NewInt(int64(21000 + i)),
			Data:     big.NewInt(int64(3000 + i)),
		}
	}

	expectedRoot := computeMerkleRootBench(transactions)
	expectedStateRoot := computeStateRootBench(big.NewInt(0), transactions)

	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = expectedRoot
	circuit.PrevStateRoot = big.NewInt(0)
	circuit.NewStateRoot = expectedStateRoot
	circuit.BatchNumber = big.NewInt(1)
	circuit.Timestamp = big.NewInt(1700000000)
	circuit.IndexerAddr = big.NewInt(9999)

	// Compile, Setup, Prove
	emptyCircuit := Circuit{}
	ccs, _ := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &emptyCircuit)
	pk, vk, _ := groth16.Setup(ccs)
	witness, _ := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	proof, _ := groth16.Prove(ccs, pk, witness)
	publicWitness, _ := witness.Public()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		err := groth16.Verify(proof, vk, publicWitness)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("Verification Time: %s", time.Since(start))
	}
}
