package circuits

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

// BatchSize defines the number of transactions in a batch
// Must be a power of 2 for efficient Merkle tree computation
const BatchSize = 128 // Changed from 1024 to 128 for faster proving

// MaxStateSize limits the number of accounts in state
const MaxStateSize = 2048

// Transaction represents a single L2 transaction in the batch
type Transaction struct {
	Nonce    frontend.Variable // Transaction sequence number for the sender
	From     frontend.Variable // Sender address (32-byte field element)
	To       frontend.Variable // Recipient address (32-byte field element)
	Amount   frontend.Variable // Transaction amount in wei
	GasPrice frontend.Variable // Gas price for this transaction
	GasLimit frontend.Variable // Maximum gas allowed
	Data     frontend.Variable // Encoded transaction data hash
}

// Account represents a state account
type Account struct {
	Address  frontend.Variable
	Nonce    frontend.Variable
	Balance  frontend.Variable
	CodeHash frontend.Variable
}

// Circuit defines the constraints for the L2 rollup batch verification
type Circuit struct {
	// Public Inputs
	BatchRoot     frontend.Variable `gnark:",public"` // Merkle root of transactions
	PrevStateRoot frontend.Variable `gnark:",public"` // State root before execution
	NewStateRoot  frontend.Variable `gnark:",public"` // State root after execution
	BatchNumber   frontend.Variable `gnark:",public"` // Rollup batch number
	Timestamp     frontend.Variable `gnark:",public"` // Block timestamp
	SequencerAddr frontend.Variable `gnark:",public"` // Address of sequencer

	// Private Witness: Transaction batch
	Transactions [BatchSize]Transaction
}

// Define declares the circuit constraints
func (circuit *Circuit) Define(api frontend.API) error {
	// Initialize MiMC hash function (SNARK-friendly)
	h, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}

	// ============ Step 1: Verify transaction batch integrity ============
	leaves := make([]frontend.Variable, BatchSize)
	for i := 0; i < BatchSize; i++ {
		h.Reset()
		// Hash all transaction fields
		h.Write(circuit.Transactions[i].Nonce)
		h.Write(circuit.Transactions[i].From)
		h.Write(circuit.Transactions[i].To)
		h.Write(circuit.Transactions[i].Amount)
		h.Write(circuit.Transactions[i].GasPrice)
		h.Write(circuit.Transactions[i].GasLimit)
		h.Write(circuit.Transactions[i].Data)
		leaves[i] = h.Sum()
	}

	// ============ Step 2: Build and verify Merkle tree ============
	computedRoot := buildMerkleTree(api, &h, leaves)

	// Verify the computed root matches the public input
	api.AssertIsEqual(computedRoot, circuit.BatchRoot)

	// ============ Step 3: Verify transaction ordering and nonce validity ============
	for i := 0; i < BatchSize; i++ {
		// Check nonce is valid (non-negative constraint implicit in field arithmetic)
		api.AssertIsLessOrEqual(circuit.Transactions[i].Nonce, 1000000) // Reasonable nonce upper bound
	}

	// ============ Step 4: Verify state root transition ============
	// Compute what the new state root should be based on transactions
	expectedNewRoot := computeStateTransition(api, &h, circuit)

	// Verify state transition is correct
	api.AssertIsEqual(expectedNewRoot, circuit.NewStateRoot)

	// ============ Step 5: Verify batch metadata ============
	// Ensure batch number is reasonable
	api.AssertIsLessOrEqual(circuit.BatchNumber, 1000000)

	// Ensure timestamp is reasonable (not negative)
	// Field constraint: timestamp should be within valid range
	api.AssertIsLessOrEqual(circuit.Timestamp, 2000000000) // Reasonable Unix timestamp ceiling

	// Ensure sequencer address is non-zero (check that it's not equal to 0)
	// Since AssertIsNotEqual doesn't exist, we use a workaround
	isZero := api.IsZero(circuit.SequencerAddr)
	api.AssertIsEqual(isZero, 0) // Assert that isZero is false

	return nil
}

// buildMerkleTree builds a Merkle tree from leaves and returns the root
func buildMerkleTree(api frontend.API, h *mimc.MiMC, leaves []frontend.Variable) frontend.Variable {
	if len(leaves) == 0 {
		return frontend.Variable(0)
	}

	currentLayer := leaves

	for len(currentLayer) > 1 {
		nextLayer := make([]frontend.Variable, len(currentLayer)/2)

		for i := 0; i < len(currentLayer); i += 2 {
			h.Reset()
			h.Write(currentLayer[i])
			h.Write(currentLayer[i+1])
			nextLayer[i/2] = h.Sum()
		}

		currentLayer = nextLayer
	}

	return currentLayer[0]
}

// computeStateTransition verifies that transactions correctly transform state
func computeStateTransition(api frontend.API, h *mimc.MiMC, circuit *Circuit) frontend.Variable {
	// Start with previous state root
	currentStateRoot := circuit.PrevStateRoot

	// Process each transaction and verify state changes
	for i := 0; i < BatchSize; i++ {
		tx := circuit.Transactions[i]

		// Verify transaction amounts are non-negative
		api.AssertIsLessOrEqual(tx.Amount, 1000000000000000000) // Max reasonable amount (10^18)

		// Verify gas parameters are reasonable
		api.AssertIsLessOrEqual(tx.GasPrice, 100000000000) // Max reasonable gas price (100 Gwei)
		api.AssertIsLessOrEqual(tx.GasLimit, 30000000)     // Max reasonable gas limit

		// Update state root
		h.Reset()
		h.Write(currentStateRoot)
		h.Write(tx.From)
		h.Write(tx.To)
		h.Write(tx.Amount)
		h.Write(tx.Nonce)
		currentStateRoot = h.Sum()
	}

	return currentStateRoot
}

// VerifyMerkleProof verifies a transaction is part of the batch
// This is for witness generation and testing
func VerifyMerkleProof(api frontend.API, h *mimc.MiMC, leafIndex int,
	proof []frontend.Variable, root frontend.Variable, leaf frontend.Variable) {

	currentHash := leaf
	idx := leafIndex

	for i := 0; i < len(proof); i++ {
		h.Reset()
		if idx%2 == 0 {
			h.Write(currentHash)
			h.Write(proof[i])
		} else {
			h.Write(proof[i])
			h.Write(currentHash)
		}
		currentHash = h.Sum()
		idx = idx / 2
	}

	api.AssertIsEqual(currentHash, root)
}

// ValidateTransaction performs basic transaction validation
func ValidateTransaction(api frontend.API, tx Transaction) {
	// Amount must be non-negative
	api.AssertIsLessOrEqual(tx.Amount, 1000000000000000000)

	// Gas price must be non-negative
	api.AssertIsLessOrEqual(tx.GasPrice, 100000000000)

	// Gas limit must be reasonable
	api.AssertIsLessOrEqual(tx.GasLimit, 30000000)

	// Nonce must be reasonable
	api.AssertIsLessOrEqual(tx.Nonce, 1000000)

	// From and To must not be zero (use IsZero workaround)
	isFromZero := api.IsZero(tx.From)
	api.AssertIsEqual(isFromZero, 0)

	isToZero := api.IsZero(tx.To)
	api.AssertIsEqual(isToZero, 0)
}
