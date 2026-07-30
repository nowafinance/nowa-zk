package circuits

import (
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
)

// SparseMerkleTree represents a sparse Merkle tree of depth D.
type SparseMerkleTree struct {
	depth      int
	leaves     map[uint64]*fr.Element
	zeroHashes []fr.Element
}

// NewSparseMerkleTree creates a new tree of the given depth.
func NewSparseMerkleTree(depth int) *SparseMerkleTree {
	smt := &SparseMerkleTree{
		depth:  depth,
		leaves: make(map[uint64]*fr.Element),
	}
	smt.computeZeroHashes()
	return smt
}

func (smt *SparseMerkleTree) computeZeroHashes() {
	smt.zeroHashes = make([]fr.Element, smt.depth+1)
	
	// zeroHashes[0] is just 0
	smt.zeroHashes[0].SetZero()

	for i := 1; i <= smt.depth; i++ {
		h := mimc.NewMiMC()
		b := smt.zeroHashes[i-1].Bytes()
		h.Write(b[:])
		h.Write(b[:])
		hashBytes := h.Sum(nil)
		smt.zeroHashes[i].SetBytes(hashBytes)
	}
}

// Update updates a leaf at the given index.
func (smt *SparseMerkleTree) Update(index uint64, value *fr.Element) {
	smt.leaves[index] = value
}

// GetPath returns the authentication path and bits for a given index.
// Path is from leaf up to root.
func (smt *SparseMerkleTree) GetPath(index uint64) ([]*big.Int, []int) {
	path := make([]*big.Int, smt.depth)
	bits := make([]int, smt.depth)

	// Since we are sparse, computing a path requires hashing up from the leaves.
	for i := 0; i < smt.depth; i++ {
		bit := int((index >> i) & 1)
		bits[i] = bit
		
		// The sibling index is index with the i-th bit flipped
		siblingIndex := index ^ (1 << i)
		
		// Get sibling value at level i
		siblingVal := smt.getNode(siblingIndex, i)
		path[i] = new(big.Int)
		siblingVal.BigInt(path[i])
	}
	
	return path, bits
}

// getNode computes the value of a node at a specific index and level (0 = leaf).
func (smt *SparseMerkleTree) getNode(index uint64, level int) fr.Element {
	if level == 0 {
		if val, ok := smt.leaves[index]; ok {
			return *val
		}
		return smt.zeroHashes[0]
	}

	// Recursively compute children
	leftChildIdx := index & ^(1 << (level - 1))
	rightChildIdx := index | (1 << (level - 1))

	left := smt.getNode(leftChildIdx, level-1)
	right := smt.getNode(rightChildIdx, level-1)

	// Hash them
	h := mimc.NewMiMC()
	lb := left.Bytes()
	rb := right.Bytes()
	h.Write(lb[:])
	h.Write(rb[:])
	
	var res fr.Element
	res.SetBytes(h.Sum(nil))
	return res
}

// Root returns the root of the tree.
func (smt *SparseMerkleTree) Root() *big.Int {
	rootFr := smt.getNode(0, smt.depth)
	res := new(big.Int)
	rootFr.BigInt(res)
	return res
}

// HashAccountLeaf computes the MiMC hash of an account state, exactly as the circuit does.
func HashAccountLeaf(index, pubX, pubY, balance, nonce *big.Int) *fr.Element {
	h := mimc.NewMiMC()
	
	var idxFr, pubXFr, pubYFr, balFr, nonceFr fr.Element
	idxFr.SetBigInt(index)
	pubXFr.SetBigInt(pubX)
	pubYFr.SetBigInt(pubY)
	balFr.SetBigInt(balance)
	nonceFr.SetBigInt(nonce)

	idxBytes := idxFr.Bytes()
	pubXBytes := pubXFr.Bytes()
	pubYBytes := pubYFr.Bytes()
	balBytes := balFr.Bytes()
	nonceBytes := nonceFr.Bytes()

	h.Write(idxBytes[:])
	h.Write(pubXBytes[:])
	h.Write(pubYBytes[:])
	h.Write(balBytes[:])
	h.Write(nonceBytes[:])

	var res fr.Element
	res.SetBytes(h.Sum(nil))
	return &res
}
