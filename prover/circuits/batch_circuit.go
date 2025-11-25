package circuits

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

const (
	TreeDepth    = 1 // A more realistic depth for testing
	BatchSize    = 1  // Start with a small batch size for development
)

// Transfer represents a single transaction in the batch.
// Note: In a real-world scenario, you'd likely have separate structs for the
// witness data vs. the circuit definition to make assignments cleaner.
type Transfer struct {
	Enabled frontend.Variable // A flag to indicate if this transfer is active or a no-op

	// from account witness
	FromIndexBits    [TreeDepth]frontend.Variable
	FromOldBalance   frontend.Variable
	FromNonce        frontend.Variable
	FromToken        frontend.Variable
	FromPathSiblings [TreeDepth]frontend.Variable

	// to account witness
	ToIndexBits    [TreeDepth]frontend.Variable
	ToOldBalance   frontend.Variable
	ToNonce        frontend.Variable
	ToToken        frontend.Variable
	ToPathSiblings [TreeDepth]frontend.Variable

	// transfer args
	Amount frontend.Variable
	Fee    frontend.Variable
}

// BatchCircuit processes a batch of transfers, chaining their state updates.
type BatchCircuit struct {
	// --- Public Inputs ---
	InitialRoot frontend.Variable `gnark:",public"`
	FinalRoot   frontend.Variable `gnark:",public"`

	// --- Private Inputs ---
	Transfers [BatchSize]Transfer
}

// Define the circuit constraints.
func (c *BatchCircuit) Define(api frontend.API) error {
	// The current state root, which gets updated after each transfer.
	currentRoot := c.InitialRoot

	// Reusable constants
	one := frontend.Variable(1)

	// Loop through each transfer in the batch.
	for i := 0; i < BatchSize; i++ {
		tx := c.Transfers[i]

		// --- 1. Sender state transition ---

		// Assert that the sender's account is part of the current state root.
		fromLeafBefore := tx.hashFromBefore(api)
		calculatedFromRoot := merkleRecompute(api, fromLeafBefore, tx.FromIndexBits, tx.FromPathSiblings)
		api.AssertIsEqual(calculatedFromRoot, currentRoot)

		// Calculate the sender's new balance and nonce.
		amtPlusFee := api.Add(tx.Amount, tx.Fee)
		fromBalanceAfter := api.Sub(tx.FromOldBalance, amtPlusFee)
		fromNonceAfter := api.Add(tx.FromNonce, one)

		// Create the new sender leaf and recompute the Merkle root.
		fromLeafAfter := tx.hashFromAfter(api, fromBalanceAfter, fromNonceAfter)
		intermediateRoot := merkleRecompute(api, fromLeafAfter, tx.FromIndexBits, tx.FromPathSiblings)

		// --- 2. Receiver state transition ---

		// Assert that the receiver's account is part of the *intermediate* state root.
		toLeafBefore := tx.hashToBefore(api)
		calculatedToRoot := merkleRecompute(api, toLeafBefore, tx.ToIndexBits, tx.ToPathSiblings)
		api.AssertIsEqual(calculatedToRoot, intermediateRoot)

		// Calculate the receiver's new balance.
		toBalanceAfter := api.Add(tx.ToOldBalance, tx.Amount)

		// Create the new receiver leaf and recompute the final root for this transaction.
		toLeafAfter := tx.hashToAfter(api, toBalanceAfter)
		nextRoot := merkleRecompute(api, toLeafAfter, tx.ToIndexBits, tx.ToPathSiblings)

		// --- 3. Conditionally update the state ---
		// If the transaction is disabled (Enabled == 0), the root remains `currentRoot`.
		// If the transaction is enabled (Enabled == 1), the root becomes `nextRoot`.
		currentRoot = api.Select(tx.Enabled, nextRoot, currentRoot)
	}

	// Finally, assert that the computed final root matches the public input.
	api.AssertIsEqual(currentRoot, c.FinalRoot)

	return nil
}

// --- Helper Functions ---

// merkleRecompute computes a Merkle root from a leaf, its path, and siblings.
func merkleRecompute(
	api frontend.API,
	leaf frontend.Variable,
	indexBits [TreeDepth]frontend.Variable,
	siblings [TreeDepth]frontend.Variable,
) frontend.Variable {
	current := leaf
	one := frontend.Variable(1)

	for i := 0; i < TreeDepth; i++ {
		bit := indexBits[i]
		api.AssertIsBoolean(bit)

		// Selects which of (current, sibling) is on the left vs. right.
		oneMinusBit := api.Sub(one, bit)
		left := api.Add(api.Mul(oneMinusBit, current), api.Mul(bit, siblings[i]))
		right := api.Add(api.Mul(oneMinusBit, siblings[i]), api.Mul(bit, current))

		current = mimcHash(api, left, right)
	}
	return current
}

// mimcHash is a helper to hash a variable number of inputs.
func mimcHash(api frontend.API, inputs ...frontend.Variable) frontend.Variable {
	h, _ := mimc.New(api) // error is negligible in-circuit
	h.Write(inputs...)
	return h.Sum()
}

// --- Per-Transfer Helper Methods ---

// indexAsField converts a bit array to a single field element.
func (t *Transfer) indexAsField(api frontend.API, bits [TreeDepth]frontend.Variable) frontend.Variable {
	res := frontend.Variable(0)
	pow := frontend.Variable(1)
	for i := 0; i < TreeDepth; i++ {
		api.AssertIsBoolean(bits[i])
		res = api.Add(res, api.Mul(bits[i], pow))
		pow = api.Mul(pow, 2)
	}
	return res
}

func (t *Transfer) hashFromBefore(api frontend.API) frontend.Variable {
	return mimcHash(api, t.indexAsField(api, t.FromIndexBits), t.FromOldBalance, t.FromNonce, t.FromToken)
}

func (t *Transfer) hashFromAfter(api frontend.API, newBalance, newNonce frontend.Variable) frontend.Variable {
	return mimcHash(api, t.indexAsField(api, t.FromIndexBits), newBalance, newNonce, t.FromToken)
}

func (t *Transfer) hashToBefore(api frontend.API) frontend.Variable {
	return mimcHash(api, t.indexAsField(api, t.ToIndexBits), t.ToOldBalance, t.ToNonce, t.ToToken)
}

func (t *Transfer) hashToAfter(api frontend.API, newBalance frontend.Variable) frontend.Variable {
	// Receiver's nonce does not change in a transfer.
	return mimcHash(api, t.indexAsField(api, t.ToIndexBits), newBalance, t.ToNonce, t.ToToken)
}
