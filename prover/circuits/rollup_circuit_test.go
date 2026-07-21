package circuits

import (
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/stretchr/testify/require"

	"os"

	mimc_offchain "github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark/constraint"
)

var (
	ccs constraint.ConstraintSystem
	pk  groth16.ProvingKey
	vk  groth16.VerifyingKey
)

func TestMain(m *testing.M) {
	// Compile circuit once
	var circuit Circuit
	var err error
	ccs, err = frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	if err != nil {
		panic("Failed to compile circuit: " + err.Error())
	}

	// Setup once
	pk, vk, err = groth16.Setup(ccs)
	if err != nil {
		panic("Failed to setup circuit: " + err.Error())
	}

	// Run tests
	os.Exit(m.Run())
}

// TestRollupCircuit tests the basic rollup circuit with valid transactions
func TestRollupCircuit(t *testing.T) {
	// Create sample transactions
	transactions := make([]Transaction, BatchSize)
	for i := 0; i < BatchSize; i++ {
		transactions[i] = Transaction{
			Nonce:    big.NewInt(int64(i)),         // Sender nonce
			From:     big.NewInt(int64(1000 + i)),  // Sender address
			To:       big.NewInt(int64(2000 + i)),  // Recipient address
			Amount:   big.NewInt(int64(100 + i)),   // Amount in wei
			GasPrice: big.NewInt(int64(20 * 1e9)),  // 20 Gwei
			GasLimit: big.NewInt(int64(21000 + i)), // Gas limit
			Data:     big.NewInt(int64(3000 + i)),  // Data hash
		}
	}

	// Compute expected Merkle root
	expectedRoot := computeMerkleRoot(t, transactions)
	t.Logf("Expected Merkle Root: %s", expectedRoot.String())

	// Compute expected state root after processing transactions
	expectedNewStateRoot := computeStateRoot(t, big.NewInt(0), transactions)

	// Create witness
	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = expectedRoot
	circuit.PrevStateRoot = big.NewInt(0)
	circuit.NewStateRoot = expectedNewStateRoot
	circuit.BatchNumber = big.NewInt(1)
	circuit.Timestamp = big.NewInt(1700000000)
	circuit.IndexerAddr = big.NewInt(9999)

	// Create witness for proving
	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	require.NoError(t, err, "Failed to create witness")

	publicWitness, err := witness.Public()
	require.NoError(t, err, "Failed to extract public witness")

	// Compile and Setup are done in TestMain

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
			Nonce:    big.NewInt(int64(i)),
			From:     big.NewInt(int64(1000 + i)),
			To:       big.NewInt(int64(2000 + i)),
			Amount:   big.NewInt(int64(100 + i)),
			GasPrice: big.NewInt(int64(20 * 1e9)),
			GasLimit: big.NewInt(int64(21000 + i)),
			Data:     big.NewInt(int64(3000 + i)),
		}
	}

	// Use wrong root
	wrongRoot := big.NewInt(12345)
	expectedNewStateRoot := computeStateRoot(t, big.NewInt(0), transactions)

	// Create witness with wrong root
	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = wrongRoot
	circuit.PrevStateRoot = big.NewInt(0)
	circuit.NewStateRoot = expectedNewStateRoot
	circuit.BatchNumber = big.NewInt(1)
	circuit.Timestamp = big.NewInt(1700000000)
	circuit.IndexerAddr = big.NewInt(9999)

	// Create witness
	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	require.NoError(t, err)

	// Compile and Setup are done in TestMain

	// Prove should fail because constraint is not satisfied
	_, err = groth16.Prove(ccs, pk, witness)
	require.Error(t, err, "Expected proof to fail with invalid root")

	t.Log("✅ Invalid root correctly rejected")
}

// TestRollupCircuitInvalidStateRoot tests that invalid state root is rejected
func TestRollupCircuitInvalidStateRoot(t *testing.T) {
	// Create sample transactions
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

	expectedBatchRoot := computeMerkleRoot(t, transactions)
	wrongStateRoot := big.NewInt(99999)

	// Create witness with wrong state root
	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = expectedBatchRoot
	circuit.PrevStateRoot = big.NewInt(0)
	circuit.NewStateRoot = wrongStateRoot
	circuit.BatchNumber = big.NewInt(1)
	circuit.Timestamp = big.NewInt(1700000000)
	circuit.IndexerAddr = big.NewInt(9999)

	// Create witness
	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	require.NoError(t, err)

	// Compile and Setup are done in TestMain

	// Prove should fail because state root constraint is not satisfied
	_, err = groth16.Prove(ccs, pk, witness)
	require.Error(t, err, "Expected proof to fail with invalid state root")

	t.Log("✅ Invalid state root correctly rejected")
}

// TestRollupCircuitEmptyBatch tests batch with all zero transactions
func TestRollupCircuitEmptyBatch(t *testing.T) {
	// Create empty transactions (all zeros)
	transactions := make([]Transaction, BatchSize)
	for i := 0; i < BatchSize; i++ {
		transactions[i] = Transaction{
			Nonce:    big.NewInt(0),
			From:     big.NewInt(0),
			To:       big.NewInt(0),
			Amount:   big.NewInt(0),
			GasPrice: big.NewInt(0),
			GasLimit: big.NewInt(0),
			Data:     big.NewInt(0),
		}
	}

	// Compute Merkle root for empty batch
	expectedRoot := computeMerkleRoot(t, transactions)
	expectedStateRoot := computeStateRoot(t, big.NewInt(0), transactions)
	t.Logf("Empty Batch Merkle Root: %s", expectedRoot.String())

	// Create witness
	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = expectedRoot
	circuit.PrevStateRoot = big.NewInt(0)
	circuit.NewStateRoot = expectedStateRoot
	circuit.BatchNumber = big.NewInt(2)
	circuit.Timestamp = big.NewInt(1700000000)
	circuit.IndexerAddr = big.NewInt(9999)

	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	require.NoError(t, err)

	publicWitness, err := witness.Public()
	require.NoError(t, err)

	// Compile and Setup are done in TestMain

	proof, err := groth16.Prove(ccs, pk, witness)
	require.NoError(t, err)

	err = groth16.Verify(proof, vk, publicWitness)
	require.NoError(t, err)

	t.Log("✅ Empty batch verified successfully")
}

// TestRollupCircuitInvalidTimestamp tests that invalid timestamp is rejected
func TestRollupCircuitInvalidTimestamp(t *testing.T) {
	// Create sample transactions
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

	expectedBatchRoot := computeMerkleRoot(t, transactions)
	expectedStateRoot := computeStateRoot(t, big.NewInt(0), transactions)
	invalidTimestamp := big.NewInt(3000000000) // Beyond reasonable range

	// Create witness with invalid timestamp
	var circuit Circuit
	copy(circuit.Transactions[:], transactions)
	circuit.BatchRoot = expectedBatchRoot
	circuit.PrevStateRoot = big.NewInt(0)
	circuit.NewStateRoot = expectedStateRoot
	circuit.BatchNumber = big.NewInt(1)
	circuit.Timestamp = invalidTimestamp
	circuit.IndexerAddr = big.NewInt(9999)

	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	require.NoError(t, err)

	// Compile and Setup are done in TestMain

	_, err = groth16.Prove(ccs, pk, witness)
	require.Error(t, err, "Expected proof to fail with invalid timestamp")

	t.Log("✅ Invalid timestamp correctly rejected")
}

// BenchmarkRollupCircuit benchmarks the circuit compilation and proving
func BenchmarkRollupCircuit(b *testing.B) {
	// Create sample transactions
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

	shift64 := new(big.Int).Lsh(big.NewInt(1), 64)
	shift128 := new(big.Int).Lsh(big.NewInt(1), 128)

	for i := 0; i < BatchSize; i++ {
		h.Reset()

		// Pack fields: Nonce + (GasLimit << 64) + (GasPrice << 128)
		nonce := transactions[i].Nonce.(*big.Int)
		gasLimit := transactions[i].GasLimit.(*big.Int)
		gasPrice := transactions[i].GasPrice.(*big.Int)

		packed := new(big.Int).Set(nonce)
		packed.Add(packed, new(big.Int).Mul(gasLimit, shift64))
		packed.Add(packed, new(big.Int).Mul(gasPrice, shift128))

		h.Write(BigIntTo32Bytes(packed))
		h.Write(BigIntTo32Bytes(transactions[i].From.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].To.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Amount.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Data.(*big.Int)))
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
	shift64 := new(big.Int).Lsh(big.NewInt(1), 64)
	shift128 := new(big.Int).Lsh(big.NewInt(1), 128)

	for i := 0; i < BatchSize; i++ {
		h.Reset()

		// Pack fields
		nonce := transactions[i].Nonce.(*big.Int)
		gasLimit := transactions[i].GasLimit.(*big.Int)
		gasPrice := transactions[i].GasPrice.(*big.Int)

		packed := new(big.Int).Set(nonce)
		packed.Add(packed, new(big.Int).Mul(gasLimit, shift64))
		packed.Add(packed, new(big.Int).Mul(gasPrice, shift128))

		h.Write(BigIntTo32Bytes(packed))
		h.Write(BigIntTo32Bytes(transactions[i].From.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].To.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Amount.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Data.(*big.Int)))
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

// Helper function to compute state root off-circuit (for testing)
func computeStateRoot(t *testing.T, prevRoot *big.Int, transactions []Transaction) *big.Int {
	h := mimc_offchain.NewMiMC()
	currentStateRoot := prevRoot

	// Process each transaction and update state root
	for i := 0; i < BatchSize; i++ {
		// 1. Compute TxHash (Leaf)
		h.Reset()

		shift64 := new(big.Int).Lsh(big.NewInt(1), 64)
		shift128 := new(big.Int).Lsh(big.NewInt(1), 128)

		nonce := transactions[i].Nonce.(*big.Int)
		gasLimit := transactions[i].GasLimit.(*big.Int)
		gasPrice := transactions[i].GasPrice.(*big.Int)

		packed := new(big.Int).Set(nonce)
		packed.Add(packed, new(big.Int).Mul(gasLimit, shift64))
		packed.Add(packed, new(big.Int).Mul(gasPrice, shift128))

		h.Write(BigIntTo32Bytes(packed))
		h.Write(BigIntTo32Bytes(transactions[i].From.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].To.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Amount.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Data.(*big.Int)))
		txHash := h.Sum(nil)

		// 2. Update State Root: Hash(StateRoot, TxHash)
		h.Reset()
		h.Write(BigIntTo32Bytes(currentStateRoot))
		h.Write(txHash)
		currentStateRoot = new(big.Int).SetBytes(h.Sum(nil))
	}

	return currentStateRoot
}

// Helper function for benchmarking state root computation
func computeStateRootBench(prevRoot *big.Int, transactions []Transaction) *big.Int {
	h := mimc_offchain.NewMiMC()
	currentStateRoot := prevRoot

	for i := 0; i < BatchSize; i++ {
		// 1. Compute TxHash (Leaf)
		h.Reset()

		shift64 := new(big.Int).Lsh(big.NewInt(1), 64)
		shift128 := new(big.Int).Lsh(big.NewInt(1), 128)

		nonce := transactions[i].Nonce.(*big.Int)
		gasLimit := transactions[i].GasLimit.(*big.Int)
		gasPrice := transactions[i].GasPrice.(*big.Int)

		packed := new(big.Int).Set(nonce)
		packed.Add(packed, new(big.Int).Mul(gasLimit, shift64))
		packed.Add(packed, new(big.Int).Mul(gasPrice, shift128))

		h.Write(BigIntTo32Bytes(packed))
		h.Write(BigIntTo32Bytes(transactions[i].From.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].To.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Amount.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Data.(*big.Int)))
		txHash := h.Sum(nil)

		// 2. Update State Root: Hash(StateRoot, TxHash)
		h.Reset()
		h.Write(BigIntTo32Bytes(currentStateRoot))
		h.Write(txHash)
		currentStateRoot = new(big.Int).SetBytes(h.Sum(nil))
	}

	return currentStateRoot
}
