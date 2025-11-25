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

// Off-circuit account representation for test setup
type Account struct {
	Index   uint64
	Balance *big.Int
	Nonce   uint64
	Token   uint64
}

func (a *Account) Leaf(t *testing.T) *big.Int {
	t.Logf("Hashing Account Leaf %d", a.Index)
	return mimcHashGo(t,
		new(big.Int).SetUint64(a.Index),
		a.Balance,
		new(big.Int).SetUint64(a.Nonce),
		new(big.Int).SetUint64(a.Token),
	)
}

func TestBatchCircuit(t *testing.T) {
	// --- Test Setup ---
	// Create some accounts
	accounts := []Account{
		{Index: 0, Balance: big.NewInt(1000), Nonce: 0, Token: 1},
		{Index: 1, Balance: big.NewInt(500), Nonce: 0, Token: 1},
		{Index: 2, Balance: big.NewInt(200), Nonce: 0, Token: 1},
		{Index: 3, Balance: big.NewInt(100), Nonce: 0, Token: 1},
	}

	// Build the initial Merkle tree
	var leaves []*big.Int
	for i := range accounts {
		leaves = append(leaves, accounts[i].Leaf(t))
	}
	tree := newSimpleMerkleTree(t, TreeDepth, leaves)
	initialRoot := tree.Root()
	t.Logf("Initial Root: %s", initialRoot)

	// --- Define the Batch ---
	// We will have one valid transfer (0 -> 1) and one disabled transfer
	var transfers [BatchSize]Transfer

	// Transfer 1: Account 0 sends 100 to Account 1 (fee 10)
	{
		fromAccount := &accounts[0]
		toAccount := &accounts[1]
		amount := big.NewInt(100)
		fee := big.NewInt(10)

		// Create pristine copies of pre-state values for the witness
		originalFromBalance := new(big.Int).Set(fromAccount.Balance)
		originalFromNonce := fromAccount.Nonce
		t.Logf("Witness FromOldBalance: %s, FromOldNonce: %d", originalFromBalance, originalFromNonce)


		// Get sender proof from the initial tree
		fromProof, fromProofBits := tree.GetProof(int(fromAccount.Index))

		// Manually apply the sender update to our off-circuit model to get the intermediate state
		fromAccount.Nonce++
		fromAccount.Balance.Sub(fromAccount.Balance, new(big.Int).Add(amount, fee))
		tree.Update(t, int(fromAccount.Index), fromAccount.Leaf(t))
		intermediateRootForDebug := tree.Root()
		t.Logf("Off-circuit Intermediate Root: %s", intermediateRootForDebug)
		
		// Get receiver proof from the INTERMEDIATE tree
		toProof, toProofBits := tree.GetProof(int(toAccount.Index))
		t.Logf("Witness ToOldBalance: %s, ToNonce: %d", toAccount.Balance, toAccount.Nonce)


		// Populate transfer data for the witness
		transfers[0].Enabled = 1
		transfers[0].Amount = amount
		transfers[0].Fee = fee
		// Sender
		transfers[0].FromIndexBits = toVariableArray(fromProofBits)
		transfers[0].FromOldBalance = originalFromBalance
		transfers[0].FromNonce = originalFromNonce
		transfers[0].FromToken = fromAccount.Token
		transfers[0].FromPathSiblings = toVariableArray(fromProof)
		// Receiver
		transfers[0].ToIndexBits = toVariableArray(toProofBits)
		transfers[0].ToOldBalance = toAccount.Balance
		transfers[0].ToNonce = toAccount.Nonce
		transfers[0].ToToken = toAccount.Token
		transfers[0].ToPathSiblings = toVariableArray(toProof)

		// Manually apply the receiver update to get the final state
		toAccount.Balance.Add(toAccount.Balance, amount)
		tree.Update(t, int(toAccount.Index), toAccount.Leaf(t))
	}

	// Transfer 2: Disabled (no-op)
	// Even disabled transfers need to have their fields initialized to non-nil values.
	transfers[1].Enabled = 0
	transfers[1].Amount = 0
	transfers[1].Fee = 0
	transfers[1].FromIndexBits = toVariableArray([]int{})
	transfers[1].FromOldBalance = 0
	transfers[1].FromNonce = 0
	transfers[1].FromToken = 0
	transfers[1].FromPathSiblings = toVariableArray([]*big.Int{})
	transfers[1].ToIndexBits = toVariableArray([]int{})
	transfers[1].ToOldBalance = 0
	transfers[1].ToNonce = 0
	transfers[1].ToToken = 0
	transfers[1].ToPathSiblings = toVariableArray([]*big.Int{})

	// --- Final state ---
	finalRoot := tree.Root()
	t.Logf("Final Root: %s", finalRoot)

	// --- Witness Assignment ---
	assignment := BatchCircuit{
		InitialRoot: initialRoot,
		FinalRoot:   finalRoot,
		Transfers:   transfers,
	}
	witness, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField())
	require.NoError(t, err)
	publicWitness, err := witness.Public()
	require.NoError(t, err)

	// --- Prove & Verify ---
	circuit := BatchCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	require.NoError(t, err, "Circuit compilation failed")

	pk, vk, err := groth16.Setup(ccs)
	require.NoError(t, err, "Trusted setup failed")

	proof, err := groth16.Prove(ccs, pk, witness)
	require.NoError(t, err, "Proof generation failed")

	err = groth16.Verify(proof, vk, publicWitness)
	require.NoError(t, err, "Proof verification failed")
}


// Converts a slice of big.Int or int to a fixed-size array of frontend.Variable
func toVariableArray[T *big.Int | int](input []T) [TreeDepth]frontend.Variable {
	var arr [TreeDepth]frontend.Variable
	for i := 0; i < TreeDepth; i++ {
		if i < len(input) {
			arr[i] = input[i]
		} else {
			arr[i] = 0 // Pad with zero if needed
		}
	}
	return arr
}


// --- Off-circuit Merkle Tree for Test Setup ---

var (
	mimcHashFunc = mimc_offchain.NewMiMC()
)

func mimcHashGo(t *testing.T, inputs ...*big.Int) *big.Int {
	mimcHashFunc.Reset()
	var logInputs []string
	for _, i := range inputs {
		mimcHashFunc.Write(i.Bytes())
		logInputs = append(logInputs, i.String())
	}
	res := new(big.Int).SetBytes(mimcHashFunc.Sum(nil))
	t.Logf("MiMC Hashing: f(%v) -> %s", logInputs, res.String())
	return res
}

type simpleMerkleTree struct {
	t      *testing.T
	nodes  [][]*big.Int
	depth  int
	leaves []*big.Int
}

func newSimpleMerkleTree(t *testing.T, depth int, leaves []*big.Int) *simpleMerkleTree {
	require.LessOrEqual(t, len(leaves), 1<<depth, "Too many leaves for the given depth")

	nodes := make([][]*big.Int, depth+1)
	nodes[0] = make([]*big.Int, 1<<depth)
	for i := 0; i < len(leaves); i++ {
		nodes[0][i] = leaves[i]
	}
	for i := len(leaves); i < 1<<depth; i++ {
		nodes[0][i] = new(big.Int) // Pad with zero leaves
	}

	for i := 0; i < depth; i++ {
		nodes[i+1] = make([]*big.Int, 1<<(depth-i-1))
		for j := 0; j < 1<<(depth-i-1); j++ {
			nodes[i+1][j] = mimcHashGo(t, nodes[i][2*j], nodes[i][2*j+1])
		}
	}

	return &simpleMerkleTree{t: t, nodes: nodes, depth: depth, leaves: leaves}
}

func (smt *simpleMerkleTree) Root() *big.Int {
	return smt.nodes[smt.depth][0]
}

func (smt *simpleMerkleTree) GetProof(leafIndex int) (pathSiblings []*big.Int, pathBits []int) {
	pathSiblings = make([]*big.Int, smt.depth)
	pathBits = make([]int, smt.depth)
	
	currentIndex := leafIndex
	for i := 0; i < smt.depth; i++ {
		pathBits[i] = currentIndex % 2
		if currentIndex%2 == 0 {
			pathSiblings[i] = smt.nodes[i][currentIndex+1] // Sibling is to the right
		} else {
			pathSiblings[i] = smt.nodes[i][currentIndex-1] // Sibling is to the left
		}
		currentIndex /= 2
	}
	return
}

func (smt *simpleMerkleTree) Update(t *testing.T, leafIndex int, newLeaf *big.Int) {
	smt.nodes[0][leafIndex] = newLeaf
	currentIndex := leafIndex
	for i := 0; i < smt.depth; i++ {
		var left, right *big.Int
		if currentIndex%2 == 0 {
			left = smt.nodes[i][currentIndex]
			right = smt.nodes[i][currentIndex+1]
		} else {
			left = smt.nodes[i][currentIndex-1]
			right = smt.nodes[i][currentIndex]
		}
		currentIndex /= 2
		smt.nodes[i+1][currentIndex] = mimcHashGo(t, left, right)
	}
}
