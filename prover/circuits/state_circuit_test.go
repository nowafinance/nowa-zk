package circuits

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/consensys/gnark-crypto/ecc/twistededwards"
	"github.com/consensys/gnark/frontend"
	gnark_twistededwards "github.com/consensys/gnark/std/algebra/native/twistededwards"
	gnark_mimc "github.com/consensys/gnark/std/hash/mimc"
	"github.com/consensys/gnark/test"
)

type SingleTransitionCircuit struct {
	OldRoot        frontend.Variable `gnark:",public"`
	NewRoot        frontend.Variable `gnark:",public"`
	WithdrawalHash frontend.Variable `gnark:",public"`
	DepositHash    frontend.Variable `gnark:",public"`
	Op             Operation
}

func (c *SingleTransitionCircuit) Define(api frontend.API) error {
	curve, _ := gnark_twistededwards.NewEdCurve(api, twistededwards.BN254)
	h, _ := gnark_mimc.NewMiMC(api)
	
	root := c.OldRoot
	wh := c.WithdrawalHash
	dh := c.DepositHash
	
	var err error
	root, wh, dh, err = processOperation(api, &c.Op, root, wh, dh, curve, &h)
	if err != nil {
		return err
	}
	
	api.AssertIsEqual(root, c.NewRoot)
	api.AssertIsEqual(wh, c.WithdrawalHash)
	api.AssertIsEqual(dh, c.DepositHash)
	return nil
}

type MemorySMT struct {
	leaves map[uint64]*big.Int
	depth  int
}

func NewMemorySMT(depth int) *MemorySMT {
	return &MemorySMT{
		leaves: make(map[uint64]*big.Int),
		depth:  depth,
	}
}

func hashGo(items ...*big.Int) *big.Int {
	h := mimc.NewMiMC()
	for _, item := range items {
		var f fr.Element
		f.SetBigInt(item)
		b := f.Bytes()
		h.Write(b[:])
	}
	res := new(big.Int)
	res.SetBytes(h.Sum(nil))
	return res
}

func (smt *MemorySMT) computeNodeHash(prefix uint64, level int, zeros []*big.Int) *big.Int {
	var matchingLeaves []uint64
	mask := ^((uint64(1) << level) - 1)
	cleanPrefix := prefix & mask
	for idx := range smt.leaves {
		if (idx & mask) == cleanPrefix {
			matchingLeaves = append(matchingLeaves, idx)
		}
	}
	
	if len(matchingLeaves) == 0 {
		return zeros[level]
	}
	if level == 0 {
		return smt.leaves[cleanPrefix]
	}
	
	leftChild := smt.computeNodeHash(cleanPrefix, level-1, zeros)
	rightChild := smt.computeNodeHash(cleanPrefix|(1<<(level-1)), level-1, zeros)
	return hashGo(leftChild, rightChild)
}

func (smt *MemorySMT) Root() *big.Int {
	zeros := make([]*big.Int, smt.depth+1)
	zeros[0] = big.NewInt(0)
	for i := 1; i <= smt.depth; i++ {
		zeros[i] = hashGo(zeros[i-1], zeros[i-1])
	}
	return smt.computeNodeHash(0, smt.depth, zeros)
}

func (smt *MemorySMT) Update(index uint64, hash *big.Int) {
	smt.leaves[index] = hash
}

func (smt *MemorySMT) GetPath(index uint64) ([MerkleDepth]*big.Int, [MerkleDepth]uint64) {
	var path [MerkleDepth]*big.Int
	var bits [MerkleDepth]uint64
	
	zeros := make([]*big.Int, smt.depth+1)
	zeros[0] = big.NewInt(0)
	for i := 1; i <= smt.depth; i++ {
		zeros[i] = hashGo(zeros[i-1], zeros[i-1])
	}
	
	for i := 0; i < smt.depth; i++ {
		bits[i] = (index >> i) & 1
		siblingIndex := index ^ (1 << i)
		path[i] = smt.computeNodeHash(siblingIndex, i, zeros)
	}
	return path, bits
}

func getPathVars(path [MerkleDepth]*big.Int, bits [MerkleDepth]uint64) ([MerkleDepth]frontend.Variable, [MerkleDepth]frontend.Variable) {
	var p [MerkleDepth]frontend.Variable
	var b [MerkleDepth]frontend.Variable
	for i := 0; i < MerkleDepth; i++ {
		p[i] = path[i]
		b[i] = bits[i]
	}
	return p, b
}

func TestStateCircuit_ValidTransfer(t *testing.T) {
	assert := test.NewAssert(t)
	smt := NewMemorySMT(MerkleDepth)

	// 1. Generate keys
	privKey, _ := eddsa.GenerateKey(rand.Reader)
	pubKey := privKey.PublicKey
	pubX := new(big.Int)
	pubKey.A.X.BigInt(pubX)
	pubY := new(big.Int)
	pubKey.A.Y.BigInt(pubY)
	
	takerPriv, _ := eddsa.GenerateKey(rand.Reader)
	takerPub := takerPriv.PublicKey
	takerX := new(big.Int)
	takerPub.A.X.BigInt(takerX)
	takerY := new(big.Int)
	takerPub.A.Y.BigInt(takerY)

	// Maker base (Index 0)
	makerBaseLeaf := hashGo(big.NewInt(0), pubX, pubY, big.NewInt(1000), big.NewInt(1))
	smt.Update(0, makerBaseLeaf)
	
	// Taker base (Index 1)
	takerBaseLeaf := hashGo(big.NewInt(1), takerX, takerY, big.NewInt(500), big.NewInt(0))
	smt.Update(1, takerBaseLeaf)

	oldRoot := smt.Root()

	// Maker paths
	makerBasePath, makerBaseBits := smt.GetPath(0)
	
	// Execute Transfer (Amount: 200)
	// Update Maker
	newMakerBaseLeaf := hashGo(big.NewInt(0), pubX, pubY, big.NewInt(800), big.NewInt(2))
	smt.Update(0, newMakerBaseLeaf)
	
	// Update Taker
	takerBasePath, takerBaseBits := smt.GetPath(1) // get path after maker update
	newTakerBaseLeaf := hashGo(big.NewInt(1), takerX, takerY, big.NewInt(700), big.NewInt(0))
	smt.Update(1, newTakerBaseLeaf)
	
	newRoot := smt.Root()
	
	// Build MsgHash and Sign
	makerMsgHashBig := hashGo(big.NewInt(1), pubX, pubY, big.NewInt(0), big.NewInt(99), big.NewInt(200), big.NewInt(0), big.NewInt(1))
	var makerMsgHashFr fr.Element
	makerMsgHashFr.SetBigInt(makerMsgHashBig)
	makerMsgHashBytes := makerMsgHashFr.Bytes()
	sig, _ := privKey.Sign(makerMsgHashBytes[:], mimc.NewMiMC())
	
	takerMsgHashBig := hashGo(big.NewInt(1), takerX, takerY, big.NewInt(1), big.NewInt(99), big.NewInt(200), big.NewInt(0), big.NewInt(0))
	var takerMsgHashFr fr.Element
	takerMsgHashFr.SetBigInt(takerMsgHashBig)
	takerMsgHashBytes := takerMsgHashFr.Bytes()
	takerSig, _ := takerPriv.Sign(takerMsgHashBytes[:], mimc.NewMiMC())

	// Build witness
	var witness SingleTransitionCircuit
	witness.OldRoot = oldRoot
	witness.NewRoot = newRoot
	witness.WithdrawalHash = 0
	witness.DepositHash = 0
	
	witness.Op.OpType = 1
	witness.Op.Amount = 200
	witness.Op.QuoteAmount = 0
	
	witness.Op.MakerPubKey.Assign(twistededwards.BN254, pubKey.Bytes())
	witness.Op.MakerSig.Assign(twistededwards.BN254, sig)
	witness.Op.MakerBase.Index = 0
	witness.Op.MakerBase.Balance = 1000
	witness.Op.MakerBase.Nonce = 1
	witness.Op.MakerBase.Path, witness.Op.MakerBase.PathBits = getPathVars(makerBasePath, makerBaseBits)
	
	// Taker
	witness.Op.TakerPubKey.Assign(twistededwards.BN254, takerPub.Bytes())
	witness.Op.TakerSig.Assign(twistededwards.BN254, takerSig)
	witness.Op.TakerBase.Index = 1
	witness.Op.TakerBase.Balance = 500
	witness.Op.TakerBase.Nonce = 0
	witness.Op.TakerBase.Path, witness.Op.TakerBase.PathBits = getPathVars(takerBasePath, takerBaseBits)
	
	// Dummy paths for inactive
	emptyPath, emptyBits := smt.GetPath(99)
	witness.Op.MakerQuote.Index = 99
	witness.Op.MakerQuote.Balance = 0
	witness.Op.MakerQuote.Nonce = 0
	witness.Op.MakerQuote.Path, witness.Op.MakerQuote.PathBits = getPathVars(emptyPath, emptyBits)
	
	witness.Op.TakerQuote.Index = 99
	witness.Op.TakerQuote.Balance = 0
	witness.Op.TakerQuote.Nonce = 0
	witness.Op.TakerQuote.Path, witness.Op.TakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

	assert.CheckCircuit(&SingleTransitionCircuit{}, test.WithValidAssignment(&witness), test.WithCurves(ecc.BN254))
}
