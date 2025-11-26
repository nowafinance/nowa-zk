package circuits

import (
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/stretchr/testify/require"

	mimc_offchain "github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
)

// TestRollupCircuit tests the basic rollup circuit with valid transactions
func TestRollupCircuit(t *testing.T) {
	// Create sample transactions
	transactions := make([]Transaction, BatchSize)
	for i := 0; i < BatchSize; i++ {
		transactions[i] = Transaction{
			From:      big.NewInt(int64(1000 + i)), // Sender address
			To:        big.NewInt(int64(2000 + i)), // Recipient address
			Amount:    big.NewInt(int64(100 + i)),  // Amount
			Nonce:     big.NewInt(int64(i)),        // Nonce
			InputHash: big.NewInt(int64(3000 + i)), // Input hash
		}
	}

	// Compute expected Merkle root
	expectedRoot := computeMerkleRoot(t, transactions)
	t.Logf("Expected Merkle Root: %s", expectedRoot.String())

	// Create witness
	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = expectedRoot
	circuit.OldStateRoot = big.NewInt(0)
	circuit.NewStateRoot = big.NewInt(1)
	circuit.BatchNumber = big.NewInt(1)

	// Create witness for proving
	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	require.NoError(t, err, "Failed to create witness")

	publicWitness, err := witness.Public()
	require.NoError(t, err, "Failed to extract public witness")

	// Compile circuit
	emptyCircuit := Circuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &emptyCircuit)
	require.NoError(t, err, "Circuit compilation failed")

	// Setup
	pk, vk, err := groth16.Setup(ccs)
	require.NoError(t, err, "Trusted setup failed")

	// Prove
	proof, err := groth16.Prove(ccs, pk, witness)
	require.NoError(t, err, "Proof generation failed")

	// Verify
	err = groth16.Verify(proof, vk, publicWitness)
	require.NoError(t, err, "Proof verification failed")

	t.Log("✅ Circuit proof verified successfully")
}

// TestRollupCircuitInvalidRoot tests that invalid root is rejected
func TestRollupCircuitInvalidRoot(t *testing.T) {
	// Create sample transactions
	transactions := make([]Transaction, BatchSize)
	for i := 0; i < BatchSize; i++ {
		transactions[i] = Transaction{
			From:      big.NewInt(int64(1000 + i)),
			To:        big.NewInt(int64(2000 + i)),
			Amount:    big.NewInt(int64(100 + i)),
			Nonce:     big.NewInt(int64(i)),
			InputHash: big.NewInt(int64(3000 + i)),
		}
	}

	// Use wrong root
	wrongRoot := big.NewInt(12345)

	// Create witness with wrong root
	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = wrongRoot
	circuit.OldStateRoot = big.NewInt(0)
	circuit.NewStateRoot = big.NewInt(1)
	circuit.BatchNumber = big.NewInt(1)

	// Create witness
	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	require.NoError(t, err)

	// Compile circuit
	emptyCircuit := Circuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &emptyCircuit)
	require.NoError(t, err)

	// Setup
	pk, _, err := groth16.Setup(ccs)
	require.NoError(t, err)

	// Prove should fail because constraint is not satisfied
	_, err = groth16.Prove(ccs, pk, witness)
	require.Error(t, err, "Expected proof to fail with invalid root")

	t.Log("✅ Invalid root correctly rejected")
}

// TestRollupCircuitEmptyBatch tests batch with all zero transactions
func TestRollupCircuitEmptyBatch(t *testing.T) {
	// Create empty transactions (all zeros)
	transactions := make([]Transaction, BatchSize)
	for i := 0; i < BatchSize; i++ {
		transactions[i] = Transaction{
			From:      big.NewInt(0),
			To:        big.NewInt(0),
			Amount:    big.NewInt(0),
			Nonce:     big.NewInt(0),
			InputHash: big.NewInt(0),
		}
	}

	// Compute Merkle root for empty batch
	expectedRoot := computeMerkleRoot(t, transactions)
	t.Logf("Empty Batch Merkle Root: %s", expectedRoot.String())

	// Create witness
	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = expectedRoot
	circuit.OldStateRoot = big.NewInt(0)
	circuit.NewStateRoot = big.NewInt(0)
	circuit.BatchNumber = big.NewInt(2)

	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	require.NoError(t, err)

	publicWitness, err := witness.Public()
	require.NoError(t, err)

	// Compile and verify
	emptyCircuit := Circuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &emptyCircuit)
	require.NoError(t, err)

	pk, vk, err := groth16.Setup(ccs)
	require.NoError(t, err)

	proof, err := groth16.Prove(ccs, pk, witness)
	require.NoError(t, err)

	err = groth16.Verify(proof, vk, publicWitness)
	require.NoError(t, err)

	t.Log("✅ Empty batch verified successfully")
}

// BenchmarkRollupCircuit benchmarks the circuit compilation and proving
func BenchmarkRollupCircuit(b *testing.B) {
	// Create sample transactions
	transactions := make([]Transaction, BatchSize)
	for i := 0; i < BatchSize; i++ {
		transactions[i] = Transaction{
			From:      big.NewInt(int64(1000 + i)),
			To:        big.NewInt(int64(2000 + i)),
			Amount:    big.NewInt(int64(100 + i)),
			Nonce:     big.NewInt(int64(i)),
			InputHash: big.NewInt(int64(3000 + i)),
		}
	}

	expectedRoot := computeMerkleRootBench(transactions)

	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = expectedRoot

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	}
}

// Helper function to convert big.Int to 32-byte slice (padded)
func BigIntTo32Bytes(i *big.Int) []byte {
	b := i.Bytes()
	if len(b) == 32 {
		return b
	}
	if len(b) > 32 {
		return b[len(b)-32:]
	}
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return padded
}

// Helper function to compute Merkle root off-circuit (for testing)
func computeMerkleRoot(t *testing.T, transactions []Transaction) *big.Int {
	h := mimc_offchain.NewMiMC()

	// Compute leaf hashes
	leaves := make([]*big.Int, BatchSize)
	for i := 0; i < BatchSize; i++ {
		h.Reset()
		h.Write(BigIntTo32Bytes(transactions[i].From.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].To.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Amount.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Nonce.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].InputHash.(*big.Int)))
		leaves[i] = new(big.Int).SetBytes(h.Sum(nil))
	}

	// Build Merkle tree
	currentLayer := leaves
	for len(currentLayer) > 1 {
		nextLayer := make([]*big.Int, len(currentLayer)/2)
		for i := 0; i < len(currentLayer); i += 2 {
			h.Reset()
			h.Write(BigIntTo32Bytes(currentLayer[i]))
			h.Write(BigIntTo32Bytes(currentLayer[i+1]))
			nextLayer[i/2] = new(big.Int).SetBytes(h.Sum(nil))
		}
		currentLayer = nextLayer
	}

	return currentLayer[0]
}

// Helper function for benchmarking (no logging)
func computeMerkleRootBench(transactions []Transaction) *big.Int {
	h := mimc_offchain.NewMiMC()

	leaves := make([]*big.Int, BatchSize)
	for i := 0; i < BatchSize; i++ {
		h.Reset()
		h.Write(BigIntTo32Bytes(transactions[i].From.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].To.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Amount.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Nonce.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].InputHash.(*big.Int)))
		leaves[i] = new(big.Int).SetBytes(h.Sum(nil))
	}

	currentLayer := leaves
	for len(currentLayer) > 1 {
		nextLayer := make([]*big.Int, len(currentLayer)/2)
		for i := 0; i < len(currentLayer); i += 2 {
			h.Reset()
			h.Write(BigIntTo32Bytes(currentLayer[i]))
			h.Write(BigIntTo32Bytes(currentLayer[i+1]))
			nextLayer[i/2] = new(big.Int).SetBytes(h.Sum(nil))
		}
		currentLayer = nextLayer
	}

	return currentLayer[0]
}
