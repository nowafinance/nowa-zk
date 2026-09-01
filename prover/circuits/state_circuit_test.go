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
	wh := frontend.Variable(0)
	dh := frontend.Variable(0)
	
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
	makerMsgHashBig := hashGo(big.NewInt(1), pubX, pubY, big.NewInt(0), big.NewInt(99))
	var makerMsgHashFr fr.Element
	makerMsgHashFr.SetBigInt(makerMsgHashBig)
	makerMsgHashBytes := makerMsgHashFr.Bytes()
	sig, _ := privKey.Sign(makerMsgHashBytes[:], mimc.NewMiMC())
	
	takerMsgHashBig := hashGo(big.NewInt(1), takerX, takerY, big.NewInt(1), big.NewInt(99))
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
	witness.Op.MakerBase.IsGenesis = 0
	witness.Op.MakerBase.Balance = 1000
	witness.Op.MakerBase.Nonce = 1
	witness.Op.MakerBase.Path, witness.Op.MakerBase.PathBits = getPathVars(makerBasePath, makerBaseBits)
	
	// Taker
	witness.Op.TakerPubKey.Assign(twistededwards.BN254, takerPub.Bytes())
	witness.Op.TakerSig.Assign(twistededwards.BN254, takerSig)
	witness.Op.TakerBase.Index = 1
	witness.Op.TakerBase.IsGenesis = 0
	witness.Op.TakerBase.Balance = 500
	witness.Op.TakerBase.Nonce = 0
	witness.Op.TakerBase.Path, witness.Op.TakerBase.PathBits = getPathVars(takerBasePath, takerBaseBits)
	
	// Dummy paths for inactive
	emptyPath, emptyBits := smt.GetPath(99)
	witness.Op.MakerQuote.Index = 99
	witness.Op.MakerQuote.IsGenesis = 0
	witness.Op.MakerQuote.Balance = 0
	witness.Op.MakerQuote.Nonce = 0
	witness.Op.MakerQuote.Path, witness.Op.MakerQuote.PathBits = getPathVars(emptyPath, emptyBits)
	
	witness.Op.TakerQuote.Index = 99
	witness.Op.TakerQuote.IsGenesis = 0
	witness.Op.TakerQuote.Balance = 0
	witness.Op.TakerQuote.Nonce = 0
	witness.Op.TakerQuote.Path, witness.Op.TakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

	assert.CheckCircuit(&SingleTransitionCircuit{}, test.WithValidAssignment(&witness), test.WithCurves(ecc.BN254))
}

func TestStateCircuit_ValidDeposit(t *testing.T) {
	assert := test.NewAssert(t)
	smt := NewMemorySMT(MerkleDepth)

	// Sequencer acts as Maker for Deposit
	seqPriv, _ := eddsa.GenerateKey(rand.Reader)
	seqPub := seqPriv.PublicKey
	seqX := new(big.Int)
	seqPub.A.X.BigInt(seqX)
	seqY := new(big.Int)
	seqPub.A.Y.BigInt(seqY)
	
	takerPriv, _ := eddsa.GenerateKey(rand.Reader)
	takerPub := takerPriv.PublicKey
	takerX := new(big.Int)
	takerPub.A.X.BigInt(takerX)
	takerY := new(big.Int)
	takerPub.A.Y.BigInt(takerY)

	// Sequencer base (Index 0) - dummy balance for deposit
	seqBaseLeaf := hashGo(big.NewInt(0), seqX, seqY, big.NewInt(0), big.NewInt(1))
	smt.Update(0, seqBaseLeaf)
	
	// Taker base (Index 1) - user balance
	takerBaseLeaf := hashGo(big.NewInt(1), takerX, takerY, big.NewInt(500), big.NewInt(0))
	smt.Update(1, takerBaseLeaf)

	oldRoot := smt.Root()

	// Maker paths
	seqBasePath, seqBaseBits := smt.GetPath(0)
	
	// Execute Deposit (Amount: 200, OpType: 3)
	// Update Taker (User gets 200)
	takerBasePath, takerBaseBits := smt.GetPath(1) // get path before update
	newTakerBaseLeaf := hashGo(big.NewInt(1), takerX, takerY, big.NewInt(700), big.NewInt(0))
	smt.Update(1, newTakerBaseLeaf)
	
	newRoot := smt.Root()
	
	// Build MsgHash and Sign for Sequencer (Maker)
	makerMsgHashBig := hashGo(big.NewInt(3), seqX, seqY, big.NewInt(0), big.NewInt(99))
	var makerMsgHashFr fr.Element
	makerMsgHashFr.SetBigInt(makerMsgHashBig)
	makerMsgHashBytes := makerMsgHashFr.Bytes()
	seqSig, _ := seqPriv.Sign(makerMsgHashBytes[:], mimc.NewMiMC())
	
	// Build MsgHash and Sign for Taker (valid signature for deposits)
	takerMsgHashBig := hashGo(big.NewInt(3), takerX, takerY, big.NewInt(1), big.NewInt(99))
	var takerMsgHashFr fr.Element
	takerMsgHashFr.SetBigInt(takerMsgHashBig)
	takerMsgHashBytes := takerMsgHashFr.Bytes()
	takerSig, _ := takerPriv.Sign(takerMsgHashBytes[:], mimc.NewMiMC())

	// Build witness
	var witness SingleTransitionCircuit
	witness.OldRoot = oldRoot
	witness.NewRoot = newRoot
	witness.WithdrawalHash = 0
	
	// Deposit hash accumulator check (starts at 0)
	depHashBig := hashGo(big.NewInt(0), big.NewInt(1), big.NewInt(200))
	witness.DepositHash = depHashBig
	
	witness.Op.OpType = 3
	witness.Op.Amount = 200
	witness.Op.QuoteAmount = 0
	
	witness.Op.MakerPubKey.Assign(twistededwards.BN254, seqPub.Bytes())
	witness.Op.MakerSig.Assign(twistededwards.BN254, seqSig)
	witness.Op.MakerBase.Index = 0
	witness.Op.MakerBase.IsGenesis = 0
	witness.Op.MakerBase.Balance = 0
	witness.Op.MakerBase.Nonce = 1
	witness.Op.MakerBase.Path, witness.Op.MakerBase.PathBits = getPathVars(seqBasePath, seqBaseBits)
	
	witness.Op.TakerPubKey.Assign(twistededwards.BN254, takerPub.Bytes())
	witness.Op.TakerSig.Assign(twistededwards.BN254, takerSig)
	witness.Op.TakerBase.Index = 1
	witness.Op.TakerBase.IsGenesis = 0
	witness.Op.TakerBase.Balance = 500
	witness.Op.TakerBase.Nonce = 0
	witness.Op.TakerBase.Path, witness.Op.TakerBase.PathBits = getPathVars(takerBasePath, takerBaseBits)
	
	// Dummy paths for inactive
	emptyPath, emptyBits := smt.GetPath(99)
	witness.Op.MakerQuote.Index = 99
	witness.Op.MakerQuote.IsGenesis = 0
	witness.Op.MakerQuote.Balance = 0
	witness.Op.MakerQuote.Nonce = 0
	witness.Op.MakerQuote.Path, witness.Op.MakerQuote.PathBits = getPathVars(emptyPath, emptyBits)
	
	witness.Op.TakerQuote.Index = 99
	witness.Op.TakerQuote.IsGenesis = 0
	witness.Op.TakerQuote.Balance = 0
	witness.Op.TakerQuote.Nonce = 0
	witness.Op.TakerQuote.Path, witness.Op.TakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

	assert.CheckCircuit(&SingleTransitionCircuit{}, test.WithValidAssignment(&witness), test.WithCurves(ecc.BN254))
}

// TestStateCircuit_ValidDeposit_AsSequencerActuallySendsIt is the real regression
// test for the deposit-signing bug: TestStateCircuit_ValidDeposit above tests a
// design that was never actually wired up (a real sequencer-signed Maker
// signature) — sequencer/cmd/sequencer/deposit_watcher.go has only ever sent an
// all-zero MakerPubKey/MakerSig/TakerSig (verified directly against that file, and
// against how prover/cmd/prover/start.go's assignEdDSASig falls back to
// R.X=R.Y=S=0 on any parse error, which a zero/wrong-length hex string hits either
// way). Before conditionalVerify, this exact witness shape failed circuit
// constraints — no deposit could ever be proven. This must now pass.
func TestStateCircuit_ValidDeposit_AsSequencerActuallySendsIt(t *testing.T) {
	assert := test.NewAssert(t)
	smt := NewMemorySMT(MerkleDepth)

	takerPriv, _ := eddsa.GenerateKey(rand.Reader)
	takerPub := takerPriv.PublicKey
	takerX := new(big.Int)
	takerPub.A.X.BigInt(takerX)
	takerY := new(big.Int)
	takerPub.A.Y.BigInt(takerY)

	// Taker base (Index 0) - user balance, starts empty (fresh account)
	takerBaseLeaf := hashGo(big.NewInt(0), takerX, takerY, big.NewInt(0), big.NewInt(0))
	smt.Update(0, takerBaseLeaf)

	oldRoot := smt.Root()

	takerBasePath, takerBaseBits := smt.GetPath(0)
	newTakerBaseLeaf := hashGo(big.NewInt(0), takerX, takerY, big.NewInt(200), big.NewInt(0))
	smt.Update(0, newTakerBaseLeaf)

	newRoot := smt.Root()

	var witness SingleTransitionCircuit
	witness.OldRoot = oldRoot
	witness.NewRoot = newRoot
	witness.WithdrawalHash = 0
	witness.DepositHash = hashGo(big.NewInt(0), big.NewInt(0), big.NewInt(200))

	witness.Op.OpType = 3 // OpDeposit
	witness.Op.Amount = 200
	witness.Op.QuoteAmount = 0

	// Maker: exactly what deposit_watcher.go sends — all zero, no real key or sig.
	witness.Op.MakerPubKey.A.X = 0
	witness.Op.MakerPubKey.A.Y = 0
	witness.Op.MakerSig.R.X = 0
	witness.Op.MakerSig.R.Y = 0
	witness.Op.MakerSig.S = 0
	emptyPath, emptyBits := smt.GetPath(99)
	witness.Op.MakerBase.Index = 99
	witness.Op.MakerBase.IsGenesis = 0
	witness.Op.MakerBase.Balance = 0
	witness.Op.MakerBase.Nonce = 0
	witness.Op.MakerBase.Path, witness.Op.MakerBase.PathBits = getPathVars(emptyPath, emptyBits)
	witness.Op.MakerQuote.Index = 99
	witness.Op.MakerQuote.IsGenesis = 0
	witness.Op.MakerQuote.Balance = 0
	witness.Op.MakerQuote.Nonce = 0
	witness.Op.MakerQuote.Path, witness.Op.MakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

	// Taker: real pubkey (the depositor), but also zero signature — deposits are
	// authenticated by the L1 Deposit event, not an off-chain signature.
	witness.Op.TakerPubKey.A.X = takerX
	witness.Op.TakerPubKey.A.Y = takerY
	witness.Op.TakerSig.R.X = 0
	witness.Op.TakerSig.R.Y = 0
	witness.Op.TakerSig.S = 0
	witness.Op.TakerBase.Index = 0
	witness.Op.TakerBase.IsGenesis = 0
	witness.Op.TakerBase.Balance = 0
	witness.Op.TakerBase.Nonce = 0
	witness.Op.TakerBase.Path, witness.Op.TakerBase.PathBits = getPathVars(takerBasePath, takerBaseBits)
	witness.Op.TakerQuote.Index = 99
	witness.Op.TakerQuote.IsGenesis = 0
	witness.Op.TakerQuote.Balance = 0
	witness.Op.TakerQuote.Nonce = 0
	witness.Op.TakerQuote.Path, witness.Op.TakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

	assert.CheckCircuit(&SingleTransitionCircuit{}, test.WithValidAssignment(&witness), test.WithCurves(ecc.BN254))
}

// TestStateCircuit_ValidDeposit_GenesisAccount is the regression test for the real
// bug found live on Sepolia (batch #1): a leaf that has NEVER been written before has
// the SMT's literal zero as its true value (see merkle_db.go's zeroHashes[0]), not
// accountLeaf(index, pubX, pubY, 0, 0) — those are different field elements for any
// real pubkey. Every other deposit test in this file pre-seeds the taker leaf via
// smt.Update() before treating it as "old" state, which never exercises this path.
// This test deliberately does NOT seed index 0 beforehand — using the tree's true
// empty-leaf path and IsGenesis=1 — reproducing exactly what a brand-new depositor's
// very first deposit looks like against a genuinely fresh Sequencer database.
func TestStateCircuit_ValidDeposit_GenesisAccount(t *testing.T) {
	assert := test.NewAssert(t)
	smt := NewMemorySMT(MerkleDepth)

	takerPriv, _ := eddsa.GenerateKey(rand.Reader)
	takerPub := takerPriv.PublicKey
	takerX := new(big.Int)
	takerPub.A.X.BigInt(takerX)
	takerY := new(big.Int)
	takerPub.A.Y.BigInt(takerY)

	// Deliberately no smt.Update(0, ...) here — index 0 has never been touched.
	oldRoot := smt.Root()

	takerBasePath, takerBaseBits := smt.GetPath(0) // true empty-tree path for index 0
	newTakerBaseLeaf := hashGo(big.NewInt(0), takerX, takerY, big.NewInt(200), big.NewInt(0))
	smt.Update(0, newTakerBaseLeaf)

	newRoot := smt.Root()

	var witness SingleTransitionCircuit
	witness.OldRoot = oldRoot
	witness.NewRoot = newRoot
	witness.WithdrawalHash = 0
	witness.DepositHash = hashGo(big.NewInt(0), big.NewInt(0), big.NewInt(200))

	witness.Op.OpType = 3 // OpDeposit
	witness.Op.Amount = 200
	witness.Op.QuoteAmount = 0

	witness.Op.MakerPubKey.A.X = 0
	witness.Op.MakerPubKey.A.Y = 0
	witness.Op.MakerSig.R.X = 0
	witness.Op.MakerSig.R.Y = 0
	witness.Op.MakerSig.S = 0
	emptyPath, emptyBits := smt.GetPath(99)
	witness.Op.MakerBase.Index = 99
	witness.Op.MakerBase.IsGenesis = 0
	witness.Op.MakerBase.Balance = 0
	witness.Op.MakerBase.Nonce = 0
	witness.Op.MakerBase.Path, witness.Op.MakerBase.PathBits = getPathVars(emptyPath, emptyBits)
	witness.Op.MakerQuote.Index = 99
	witness.Op.MakerQuote.IsGenesis = 0
	witness.Op.MakerQuote.Balance = 0
	witness.Op.MakerQuote.Nonce = 0
	witness.Op.MakerQuote.Path, witness.Op.MakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

	witness.Op.TakerPubKey.A.X = takerX
	witness.Op.TakerPubKey.A.Y = takerY
	witness.Op.TakerSig.R.X = 0
	witness.Op.TakerSig.R.Y = 0
	witness.Op.TakerSig.S = 0
	witness.Op.TakerBase.Index = 0
	witness.Op.TakerBase.IsGenesis = 1 // <-- the property under test
	witness.Op.TakerBase.Balance = 0
	witness.Op.TakerBase.Nonce = 0
	witness.Op.TakerBase.Path, witness.Op.TakerBase.PathBits = getPathVars(takerBasePath, takerBaseBits)
	witness.Op.TakerQuote.Index = 99
	witness.Op.TakerQuote.IsGenesis = 0
	witness.Op.TakerQuote.Balance = 0
	witness.Op.TakerQuote.Nonce = 0
	witness.Op.TakerQuote.Path, witness.Op.TakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

	assert.CheckCircuit(&SingleTransitionCircuit{}, test.WithValidAssignment(&witness), test.WithCurves(ecc.BN254))
}

// TestStateCircuit_GenesisCannotForgeBalance guards the security property IsGenesis
// introduces: a leg claiming IsGenesis=1 MUST also claim Balance=0 and Nonce=0 — it
// cannot use the genesis flag to skip the inclusion check while asserting a
// fabricated non-zero starting balance out of thin air.
func TestStateCircuit_GenesisCannotForgeBalance(t *testing.T) {
	assert := test.NewAssert(t)
	smt := NewMemorySMT(MerkleDepth)

	takerPriv, _ := eddsa.GenerateKey(rand.Reader)
	takerPub := takerPriv.PublicKey
	takerX := new(big.Int)
	takerPub.A.X.BigInt(takerX)
	takerY := new(big.Int)
	takerPub.A.Y.BigInt(takerY)

	oldRoot := smt.Root() // index 0 never touched
	takerBasePath, takerBaseBits := smt.GetPath(0)
	newTakerBaseLeaf := hashGo(big.NewInt(0), takerX, takerY, big.NewInt(1_000_200), big.NewInt(0))
	smt.Update(0, newTakerBaseLeaf)
	newRoot := smt.Root()

	var witness SingleTransitionCircuit
	witness.OldRoot = oldRoot
	witness.NewRoot = newRoot
	witness.WithdrawalHash = 0
	witness.DepositHash = hashGo(big.NewInt(0), big.NewInt(0), big.NewInt(200))

	witness.Op.OpType = 3
	witness.Op.Amount = 200
	witness.Op.QuoteAmount = 0

	witness.Op.MakerPubKey.A.X = 0
	witness.Op.MakerPubKey.A.Y = 0
	witness.Op.MakerSig.R.X = 0
	witness.Op.MakerSig.R.Y = 0
	witness.Op.MakerSig.S = 0
	emptyPath, emptyBits := smt.GetPath(99)
	witness.Op.MakerBase.Index = 99
	witness.Op.MakerBase.IsGenesis = 0
	witness.Op.MakerBase.Balance = 0
	witness.Op.MakerBase.Nonce = 0
	witness.Op.MakerBase.Path, witness.Op.MakerBase.PathBits = getPathVars(emptyPath, emptyBits)
	witness.Op.MakerQuote.Index = 99
	witness.Op.MakerQuote.IsGenesis = 0
	witness.Op.MakerQuote.Balance = 0
	witness.Op.MakerQuote.Nonce = 0
	witness.Op.MakerQuote.Path, witness.Op.MakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

	witness.Op.TakerPubKey.A.X = takerX
	witness.Op.TakerPubKey.A.Y = takerY
	witness.Op.TakerSig.R.X = 0
	witness.Op.TakerSig.R.Y = 0
	witness.Op.TakerSig.S = 0
	witness.Op.TakerBase.Index = 0
	witness.Op.TakerBase.IsGenesis = 1
	// Fabricated: claims a 1,000,000 head start on top of genesis, which IsGenesis
	// must reject regardless of what path is supplied.
	witness.Op.TakerBase.Balance = 1_000_000
	witness.Op.TakerBase.Nonce = 0
	witness.Op.TakerBase.Path, witness.Op.TakerBase.PathBits = getPathVars(takerBasePath, takerBaseBits)
	witness.Op.TakerQuote.Index = 99
	witness.Op.TakerQuote.IsGenesis = 0
	witness.Op.TakerQuote.Balance = 0
	witness.Op.TakerQuote.Nonce = 0
	witness.Op.TakerQuote.Path, witness.Op.TakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

	assert.CheckCircuit(&SingleTransitionCircuit{}, test.WithInvalidAssignment(&witness), test.WithCurves(ecc.BN254))
}

// TestStateCircuit_TradeStillRequiresRealTakerSignature guards against
// conditionalVerify accidentally weakening Trade: only Deposit/Transfer/Withdrawal
// should ever bypass the Taker check. A Trade with a genuine Maker signature but an
// all-zero Taker signature must still be rejected.
func TestStateCircuit_TradeStillRequiresRealTakerSignature(t *testing.T) {
	assert := test.NewAssert(t)
	smt := NewMemorySMT(MerkleDepth)

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

	makerBaseLeaf := hashGo(big.NewInt(0), pubX, pubY, big.NewInt(1000), big.NewInt(1))
	smt.Update(0, makerBaseLeaf)
	takerQuoteLeaf := hashGo(big.NewInt(1), takerX, takerY, big.NewInt(5000), big.NewInt(0))
	smt.Update(1, takerQuoteLeaf)

	oldRoot := smt.Root()
	makerBasePath, makerBaseBits := smt.GetPath(0)

	newMakerBaseLeaf := hashGo(big.NewInt(0), pubX, pubY, big.NewInt(800), big.NewInt(1))
	smt.Update(0, newMakerBaseLeaf)
	takerQuotePath, takerQuoteBits := smt.GetPath(1)
	newTakerQuoteLeaf := hashGo(big.NewInt(1), takerX, takerY, big.NewInt(4000), big.NewInt(0))
	smt.Update(1, newTakerQuoteLeaf)

	newRoot := smt.Root()

	// Real, valid Maker signature (a real trade order).
	makerMsgHashBig := hashGo(big.NewInt(0), pubX, pubY, big.NewInt(0), big.NewInt(99))
	var makerMsgHashFr fr.Element
	makerMsgHashFr.SetBigInt(makerMsgHashBig)
	makerMsgHashBytes := makerMsgHashFr.Bytes()
	sig, _ := privKey.Sign(makerMsgHashBytes[:], mimc.NewMiMC())

	var witness SingleTransitionCircuit
	witness.OldRoot = oldRoot
	witness.NewRoot = newRoot
	witness.WithdrawalHash = 0
	witness.DepositHash = 0

	witness.Op.OpType = 0 // OpTrade
	witness.Op.Amount = 200
	witness.Op.QuoteAmount = 1000

	witness.Op.MakerPubKey.Assign(twistededwards.BN254, pubKey.Bytes())
	witness.Op.MakerSig.Assign(twistededwards.BN254, sig)
	witness.Op.MakerBase.Index = 0
	witness.Op.MakerBase.IsGenesis = 0
	witness.Op.MakerBase.Balance = 1000
	witness.Op.MakerBase.Nonce = 1
	witness.Op.MakerBase.Path, witness.Op.MakerBase.PathBits = getPathVars(makerBasePath, makerBaseBits)
	emptyPath, emptyBits := smt.GetPath(99)
	witness.Op.MakerQuote.Index = 99
	witness.Op.MakerQuote.IsGenesis = 0
	witness.Op.MakerQuote.Balance = 0
	witness.Op.MakerQuote.Nonce = 0
	witness.Op.MakerQuote.Path, witness.Op.MakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

	// Taker: real pubkey, but a bogus all-zero signature — must be rejected, Trade
	// always needs a genuine Taker signature.
	witness.Op.TakerPubKey.A.X = takerX
	witness.Op.TakerPubKey.A.Y = takerY
	witness.Op.TakerSig.R.X = 0
	witness.Op.TakerSig.R.Y = 0
	witness.Op.TakerSig.S = 0
	witness.Op.TakerBase.Index = 99
	witness.Op.TakerBase.IsGenesis = 0
	witness.Op.TakerBase.Balance = 0
	witness.Op.TakerBase.Nonce = 0
	witness.Op.TakerBase.Path, witness.Op.TakerBase.PathBits = getPathVars(emptyPath, emptyBits)
	witness.Op.TakerQuote.Index = 1
	witness.Op.TakerQuote.IsGenesis = 0
	witness.Op.TakerQuote.Balance = 5000
	witness.Op.TakerQuote.Nonce = 0
	witness.Op.TakerQuote.Path, witness.Op.TakerQuote.PathBits = getPathVars(takerQuotePath, takerQuoteBits)

	assert.CheckCircuit(&SingleTransitionCircuit{}, test.WithInvalidAssignment(&witness), test.WithCurves(ecc.BN254))
}

// TestStateTransitionCircuit_FullBatchOf25 is the first real test of
// StateTransitionCircuit.Define()'s multi-op loop — every other test in this file
// exercises processOperation() directly via the single-op SingleTransitionCircuit
// wrapper, which never touched the real [BatchSize]Operation array or the root/hash
// threading between consecutive ops in one proof. With BatchSize raised from 1 to
// 25, this is the path every real batch now actually takes, so it needs its own
// dedicated coverage, not an inference from the single-op tests passing.
//
// 25 sequential real transfers between the same two accounts, applied one after
// another to both the reference tree and the witness — mirrors exactly how a real
// Sequencer batch accumulates fills: each op's "old" snapshot must reflect every
// prior op in the same batch, not the batch's starting state.
func TestStateTransitionCircuit_FullBatchOf25(t *testing.T) {
	assert := test.NewAssert(t)
	smt := NewMemorySMT(MerkleDepth)

	senderPriv, _ := eddsa.GenerateKey(rand.Reader)
	senderPub := senderPriv.PublicKey
	senderX := new(big.Int)
	senderPub.A.X.BigInt(senderX)
	senderY := new(big.Int)
	senderPub.A.Y.BigInt(senderY)

	receiverPriv, _ := eddsa.GenerateKey(rand.Reader)
	receiverPub := receiverPriv.PublicKey
	receiverX := new(big.Int)
	receiverPub.A.X.BigInt(receiverX)
	receiverY := new(big.Int)
	receiverPub.A.Y.BigInt(receiverY)

	const transferAmount = 100
	senderBalance := int64(25000)
	senderNonce := int64(0)
	receiverBalance := int64(0)

	smt.Update(0, hashGo(big.NewInt(0), senderX, senderY, big.NewInt(senderBalance), big.NewInt(senderNonce)))
	smt.Update(1, hashGo(big.NewInt(1), receiverX, receiverY, big.NewInt(receiverBalance), big.NewInt(0)))

	var witness StateTransitionCircuit
	witness.OldRoot = smt.Root()
	witness.WithdrawalHash = 0
	witness.DepositHash = 0

	// Inactive-leg dummy path — safe to compute once: MakerQuote/TakerQuote are
	// inactive for every op here (Transfer), so their computed root is discarded via
	// api.Select regardless of whether this path is still "fresh" after later tree
	// updates elsewhere. Same established pattern the single-op tests above use.
	emptyPath, emptyBits := smt.GetPath(99)

	if BatchSize != 25 {
		t.Fatalf("this test assumes BatchSize == 25 to match the account seeding above; got %d", BatchSize)
	}

	for i := 0; i < BatchSize; i++ {
		witness.Ops[i].OpType = 1 // OpTransfer
		witness.Ops[i].Amount = transferAmount
		witness.Ops[i].QuoteAmount = 0

		witness.Ops[i].MakerPubKey.Assign(twistededwards.BN254, senderPub.Bytes())
		msgHashBig := hashGo(big.NewInt(1), senderX, senderY, big.NewInt(0), big.NewInt(99))
		var msgHashFr fr.Element
		msgHashFr.SetBigInt(msgHashBig)
		msgHashBytes := msgHashFr.Bytes()
		sig, _ := senderPriv.Sign(msgHashBytes[:], mimc.NewMiMC())
		witness.Ops[i].MakerSig.Assign(twistededwards.BN254, sig)

		// Snapshot MakerBase (sender), then apply its mutation immediately — the
		// circuit processes this leg, and updates root, BEFORE it even looks at
		// TakerBase's inclusion proof. Indices 0 and 1 are adjacent (siblings at the
		// tree's bottom level), so the receiver's own Merkle path genuinely changes
		// the instant the sender's leaf does — fetching both paths upfront (as an
		// earlier version of this test did) reads a stale receiver path and fails.
		senderPath, senderBits := smt.GetPath(0)
		witness.Ops[i].MakerBase.Index = 0
		witness.Ops[i].MakerBase.IsGenesis = 0
		witness.Ops[i].MakerBase.Balance = senderBalance
		witness.Ops[i].MakerBase.Nonce = senderNonce
		witness.Ops[i].MakerBase.Path, witness.Ops[i].MakerBase.PathBits = getPathVars(senderPath, senderBits)

		senderBalance -= transferAmount
		senderNonce++
		smt.Update(0, hashGo(big.NewInt(0), senderX, senderY, big.NewInt(senderBalance), big.NewInt(senderNonce)))

		witness.Ops[i].MakerQuote.Index = 99
		witness.Ops[i].MakerQuote.IsGenesis = 0
		witness.Ops[i].MakerQuote.Balance = 0
		witness.Ops[i].MakerQuote.Nonce = 0
		witness.Ops[i].MakerQuote.Path, witness.Ops[i].MakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

		// Taker signature is never checked for Transfer (conditionalVerify only
		// requires it for Trade) — left zero on purpose, matching how a real
		// Transfer transition is actually constructed.
		witness.Ops[i].TakerPubKey.Assign(twistededwards.BN254, receiverPub.Bytes())
		witness.Ops[i].TakerSig.R.X = 0
		witness.Ops[i].TakerSig.R.Y = 0
		witness.Ops[i].TakerSig.S = 0

		// Snapshot TakerBase (receiver) only now — after MakerBase's mutation above,
		// matching the circuit's actual processing order.
		receiverPath, receiverBits := smt.GetPath(1)
		witness.Ops[i].TakerBase.Index = 1
		witness.Ops[i].TakerBase.IsGenesis = 0
		witness.Ops[i].TakerBase.Balance = receiverBalance
		witness.Ops[i].TakerBase.Nonce = 0
		witness.Ops[i].TakerBase.Path, witness.Ops[i].TakerBase.PathBits = getPathVars(receiverPath, receiverBits)

		receiverBalance += transferAmount
		smt.Update(1, hashGo(big.NewInt(1), receiverX, receiverY, big.NewInt(receiverBalance), big.NewInt(0)))

		witness.Ops[i].TakerQuote.Index = 99
		witness.Ops[i].TakerQuote.IsGenesis = 0
		witness.Ops[i].TakerQuote.Balance = 0
		witness.Ops[i].TakerQuote.Nonce = 0
		witness.Ops[i].TakerQuote.Path, witness.Ops[i].TakerQuote.PathBits = getPathVars(emptyPath, emptyBits)
	}

	witness.NewRoot = smt.Root()

	assert.CheckCircuit(&StateTransitionCircuit{}, test.WithValidAssignment(&witness), test.WithCurves(ecc.BN254))

	if senderBalance != 25000-25*transferAmount || receiverBalance != 25*transferAmount {
		t.Fatalf("test bookkeeping bug (not a circuit bug): sender=%d receiver=%d", senderBalance, receiverBalance)
	}
}

func TestStateCircuit_InvalidSignature(t *testing.T) {
	assert := test.NewAssert(t)
	smt := NewMemorySMT(MerkleDepth)

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

	makerBaseLeaf := hashGo(big.NewInt(0), pubX, pubY, big.NewInt(1000), big.NewInt(1))
	smt.Update(0, makerBaseLeaf)
	
	takerBaseLeaf := hashGo(big.NewInt(1), takerX, takerY, big.NewInt(500), big.NewInt(0))
	smt.Update(1, takerBaseLeaf)

	oldRoot := smt.Root()

	makerBasePath, makerBaseBits := smt.GetPath(0)
	
	newMakerBaseLeaf := hashGo(big.NewInt(0), pubX, pubY, big.NewInt(800), big.NewInt(2))
	smt.Update(0, newMakerBaseLeaf)
	
	takerBasePath, takerBaseBits := smt.GetPath(1)
	newTakerBaseLeaf := hashGo(big.NewInt(1), takerX, takerY, big.NewInt(700), big.NewInt(0))
	smt.Update(1, newTakerBaseLeaf)
	
	newRoot := smt.Root()
	
	// Generate INVALID signature (sign different amount)
	makerMsgHashBig := hashGo(big.NewInt(1), pubX, pubY, big.NewInt(0), big.NewInt(99))
	var makerMsgHashFr fr.Element
	makerMsgHashFr.SetBigInt(makerMsgHashBig)
	makerMsgHashBytes := makerMsgHashFr.Bytes()
	// Sign a DIFFERENT payload so verification fails against the circuit message.
	invalidMsg := hashGo(big.NewInt(1), pubX, pubY, big.NewInt(0), big.NewInt(98))
	var invalidFr fr.Element
	invalidFr.SetBigInt(invalidMsg)
	invalidBytes := invalidFr.Bytes()
	invalidSig, _ := privKey.Sign(invalidBytes[:], mimc.NewMiMC())
	_ = makerMsgHashBytes
	
	takerMsgHashBig := hashGo(big.NewInt(1), takerX, takerY, big.NewInt(1), big.NewInt(99))
	var takerMsgHashFr fr.Element
	takerMsgHashFr.SetBigInt(takerMsgHashBig)
	takerMsgHashBytes := takerMsgHashFr.Bytes()
	takerSig, _ := takerPriv.Sign(takerMsgHashBytes[:], mimc.NewMiMC())

	var witness SingleTransitionCircuit
	witness.OldRoot = oldRoot
	witness.NewRoot = newRoot
	witness.WithdrawalHash = 0
	witness.DepositHash = 0
	
	witness.Op.OpType = 1
	witness.Op.Amount = 200 // Circuit enforces Amount = 200, but Sig is for 99999!
	witness.Op.QuoteAmount = 0
	
	witness.Op.MakerPubKey.Assign(twistededwards.BN254, pubKey.Bytes())
	witness.Op.MakerSig.Assign(twistededwards.BN254, invalidSig) // INVALID SIG
	witness.Op.MakerBase.Index = 0
	witness.Op.MakerBase.IsGenesis = 0
	witness.Op.MakerBase.Balance = 1000
	witness.Op.MakerBase.Nonce = 1
	witness.Op.MakerBase.Path, witness.Op.MakerBase.PathBits = getPathVars(makerBasePath, makerBaseBits)
	
	witness.Op.TakerPubKey.Assign(twistededwards.BN254, takerPub.Bytes())
	witness.Op.TakerSig.Assign(twistededwards.BN254, takerSig)
	witness.Op.TakerBase.Index = 1
	witness.Op.TakerBase.IsGenesis = 0
	witness.Op.TakerBase.Balance = 500
	witness.Op.TakerBase.Nonce = 0
	witness.Op.TakerBase.Path, witness.Op.TakerBase.PathBits = getPathVars(takerBasePath, takerBaseBits)
	
	emptyPath, emptyBits := smt.GetPath(99)
	witness.Op.MakerQuote.Index = 99
	witness.Op.MakerQuote.IsGenesis = 0
	witness.Op.MakerQuote.Balance = 0
	witness.Op.MakerQuote.Nonce = 0
	witness.Op.MakerQuote.Path, witness.Op.MakerQuote.PathBits = getPathVars(emptyPath, emptyBits)
	
	witness.Op.TakerQuote.Index = 99
	witness.Op.TakerQuote.IsGenesis = 0
	witness.Op.TakerQuote.Balance = 0
	witness.Op.TakerQuote.Nonce = 0
	witness.Op.TakerQuote.Path, witness.Op.TakerQuote.PathBits = getPathVars(emptyPath, emptyBits)

	assert.CheckCircuit(&SingleTransitionCircuit{}, test.WithInvalidAssignment(&witness), test.WithCurves(ecc.BN254))
}

