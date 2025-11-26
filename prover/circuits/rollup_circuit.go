package circuits

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

// BatchSize defines the number of transactions in a batch
// Must be a power of 2 for efficient Merkle tree computation
const BatchSize = 128 // Changed to power of 2 for optimal tree depth

// Transaction represents a single L2 transaction in the batch.
type Transaction struct {
	From      frontend.Variable // Sender address (field element)
	To        frontend.Variable // Recipient address (field element)
	Amount    frontend.Variable // Transaction amount
	Nonce     frontend.Variable // Sender's nonce
	InputHash frontend.Variable // Hash of transaction input data
}

// Circuit defines the constraints for the L2 rollup batch verification.
// It proves that a batch of transactions produces a specific Merkle root.
type Circuit struct {
	// Public Inputs
	BatchRoot    frontend.Variable `gnark:",public"` // Merkle root of the batch
	OldStateRoot frontend.Variable `gnark:",public"` // State root before batch execution
	NewStateRoot frontend.Variable `gnark:",public"` // State root after batch execution
	BatchNumber  frontend.Variable `gnark:",public"` // Batch number

	// Private Witness: The batch of transactions
	Transactions [BatchSize]Transaction
}

// Define declares the circuit constraints.
// It computes a Merkle tree from transaction hashes and verifies the root.
func (circuit *Circuit) Define(api frontend.API) error {
	// Initialize MiMC hash function (SNARK-friendly)
	h, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}

	// Step 1: Compute leaf hashes (hash each transaction)
	leaves := make([]frontend.Variable, BatchSize)
	for i := 0; i < BatchSize; i++ {
		h.Reset()
		h.Write(circuit.Transactions[i].From)
		h.Write(circuit.Transactions[i].To)
		h.Write(circuit.Transactions[i].Amount)
		h.Write(circuit.Transactions[i].Nonce)
		h.Write(circuit.Transactions[i].InputHash)
		leaves[i] = h.Sum()
	}

	// Debug: Print first leaf
	api.Println("DEBUG: Circuit Leaf 0:", leaves[0])
	api.Println("DEBUG: Circuit Tx 0 InputHash:", circuit.Transactions[0].InputHash)

	// Step 2: Build Merkle tree bottom-up
	// Since BatchSize is a power of 2, we can use a perfect binary tree
	currentLayer := leaves

	// Build tree layer by layer until we reach the root
	for len(currentLayer) > 1 {
		nextLayer := make([]frontend.Variable, len(currentLayer)/2)

		// Hash pairs of nodes to create parent layer
		for i := 0; i < len(currentLayer); i += 2 {
			h.Reset()
			// Hash left child and right child
			h.Write(currentLayer[i])
			h.Write(currentLayer[i+1])
			nextLayer[i/2] = h.Sum()
		}

		currentLayer = nextLayer
	}

	// Step 3: Verify computed root matches the public input
	api.AssertIsEqual(circuit.BatchRoot, currentLayer[0])

	// Note: In a real rollup, we would verify that processing the transactions
	// transforms OldStateRoot to NewStateRoot. For this version, we just
	// expose them as public inputs so the contract can validate them.
	// We add dummy constraints to ensure they are part of the circuit.
	api.AssertIsEqual(circuit.OldStateRoot, circuit.OldStateRoot)
	api.AssertIsEqual(circuit.NewStateRoot, circuit.NewStateRoot)
	api.AssertIsEqual(circuit.BatchNumber, circuit.BatchNumber)

	return nil
}

// Helper function to hash a transaction (can be used for witness generation)
// Note: This is not part of the circuit, just a helper for tests
func HashTransaction(h *mimc.MiMC, tx Transaction) frontend.Variable {
	h.Reset()
	h.Write(tx.From)
	h.Write(tx.To)
	h.Write(tx.Amount)
	h.Write(tx.Nonce)
	h.Write(tx.InputHash)
	return h.Sum()
}
