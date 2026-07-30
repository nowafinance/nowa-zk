package circuits

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/twistededwards"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/test"
)

func TestStateTransitionCircuitCompilation(t *testing.T) {
	assert := test.NewAssert(t)
	var circuit StateTransitionCircuit

	// Just checking if the circuit compiles and is valid structurally.
	_, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	assert.NoError(err)
}

func bytesToBigInt(b [32]byte) *big.Int {
	return new(big.Int).SetBytes(b[:])
}

func TestValidTransfer(t *testing.T) {
	assert := test.NewAssert(t)

	// 1. Initialize our Go Merkle Tree
	tree := NewSparseMerkleTree(MerkleDepth)

	// 2. Create Alice (Sender) with BabyJubJub EdDSA
	alicePriv, _ := eddsa.GenerateKey(rand.Reader)
	alicePub := alicePriv.PublicKey
	aliceIndex := uint64(1)
	aliceBal := big.NewInt(1000)
	aliceNonce := big.NewInt(0)
	
	aliceLeaf := HashAccountLeaf(big.NewInt(int64(aliceIndex)), bytesToBigInt(alicePub.A.X.Bytes()), bytesToBigInt(alicePub.A.Y.Bytes()), aliceBal, aliceNonce)
	tree.Update(aliceIndex, aliceLeaf)

	// 3. Create Bob (Receiver) with BabyJubJub EdDSA
	bobPriv, _ := eddsa.GenerateKey(rand.Reader)
	bobPub := bobPriv.PublicKey
	bobIndex := uint64(2)
	bobBal := big.NewInt(500)
	bobNonce := big.NewInt(0)
	
	bobLeaf := HashAccountLeaf(big.NewInt(int64(bobIndex)), bytesToBigInt(bobPub.A.X.Bytes()), bytesToBigInt(bobPub.A.Y.Bytes()), bobBal, bobNonce)
	tree.Update(bobIndex, bobLeaf)

	oldRoot := tree.Root()

	// 4. Create the Transfer (Alice sends 100 to Bob)
	amount := big.NewInt(100)
	
	msg := make([]byte, 32)
	rand.Read(msg) // Dummy message hash for the signature
	msg[0] = 0 // Ensure < curve order
	msgInt := new(big.Int).SetBytes(msg)

	hFunc := mimc.NewMiMC()
	sigBin, _ := alicePriv.Sign(msg, hFunc)

	// Get Merkle proofs for Alice
	alicePath, aliceBits := tree.GetPath(aliceIndex)

	// Update Alice's state in the tree
	newAliceBal := new(big.Int).Sub(aliceBal, amount)
	newAliceNonce := new(big.Int).Add(aliceNonce, big.NewInt(1))
	newAliceLeaf := HashAccountLeaf(big.NewInt(int64(aliceIndex)), bytesToBigInt(alicePub.A.X.Bytes()), bytesToBigInt(alicePub.A.Y.Bytes()), newAliceBal, newAliceNonce)
	tree.Update(aliceIndex, newAliceLeaf)

	// Get Merkle proofs for Bob (Using the tree AFTER Alice's update)
	bobPath, bobBits := tree.GetPath(bobIndex)

	// Update Bob's state in the tree
	newBobBal := new(big.Int).Add(bobBal, amount)
	newBobLeaf := HashAccountLeaf(big.NewInt(int64(bobIndex)), bytesToBigInt(bobPub.A.X.Bytes()), bytesToBigInt(bobPub.A.Y.Bytes()), newBobBal, bobNonce)
	tree.Update(bobIndex, newBobLeaf)

	newRoot := tree.Root()

	// 5. Construct the Witness
	var witness StateTransitionCircuit
	witness.OldRoot = oldRoot
	witness.NewRoot = newRoot
	witness.WithdrawalHash = 0 // No withdrawals in this batch

	// Fill the first operation
	witness.Ops[0] = Operation{
		OpType:          OpTransfer,
		MessageHash:     msgInt,
		SenderIndex:     aliceIndex,
		SenderBalance:   aliceBal,
		SenderNonce:     aliceNonce,
		Amount:          amount,
		ReceiverIndex:   bobIndex,
		ReceiverBalance: bobBal,
		ReceiverNonce:   bobNonce,
	}
	
	witness.Ops[0].PubKey.Assign(twistededwards.BN254, alicePub.Bytes())
	witness.Ops[0].Sig.Assign(twistededwards.BN254, sigBin)
	witness.Ops[0].ReceiverPubKey.Assign(twistededwards.BN254, bobPub.Bytes())

	for i := 0; i < MerkleDepth; i++ {
		witness.Ops[0].SenderPath[i] = alicePath[i]
		witness.Ops[0].SenderPathBits[i] = aliceBits[i]
		witness.Ops[0].ReceiverPath[i] = bobPath[i]
		witness.Ops[0].ReceiverPathBits[i] = bobBits[i]
	}

	// Fill the rest with dummy NO-OPs (Transfer 0 from Bob to Alice to pass validation)
	currentBobNonce := bobNonce
	currentAliceBal := newAliceBal
	currentAliceNonce := newAliceNonce

	for i := 1; i < BatchSize; i++ {
		msgDummy := make([]byte, 32)
		rand.Read(msgDummy)
		msgDummy[0] = 0
		msgDummyInt := new(big.Int).SetBytes(msgDummy)

		hFuncDummy := mimc.NewMiMC()
		sigBinDummy, _ := bobPriv.Sign(msgDummy, hFuncDummy)

		// 1. Get Sender (Bob) Path
		bPath, bBits := tree.GetPath(bobIndex)
		
		// 2. Simulate Sender Update in Go Tree
		currentBobNonce = new(big.Int).Add(currentBobNonce, big.NewInt(1))
		bobLeafDummy := HashAccountLeaf(big.NewInt(int64(bobIndex)), bytesToBigInt(bobPub.A.X.Bytes()), bytesToBigInt(bobPub.A.Y.Bytes()), newBobBal, currentBobNonce)
		tree.Update(bobIndex, bobLeafDummy)

		// 3. Get Receiver (Alice) Path (from tree AFTER Sender update)
		aPath, aBits := tree.GetPath(aliceIndex)

		witness.Ops[i] = Operation{
			OpType:          OpTransfer,
			MessageHash:     msgDummyInt,
			SenderIndex:     bobIndex,
			SenderBalance:   newBobBal,
			SenderNonce:     new(big.Int).Sub(currentBobNonce, big.NewInt(1)), // The nonce BEFORE the update
			Amount:          0,
			ReceiverIndex:   aliceIndex,
			ReceiverBalance: currentAliceBal,
			ReceiverNonce:   currentAliceNonce,
		}
		
		witness.Ops[i].PubKey.Assign(twistededwards.BN254, bobPub.Bytes())
		witness.Ops[i].Sig.Assign(twistededwards.BN254, sigBinDummy)
		witness.Ops[i].ReceiverPubKey.Assign(twistededwards.BN254, alicePub.Bytes())
		
		for j := 0; j < MerkleDepth; j++ {
			witness.Ops[i].SenderPath[j] = bPath[j]
			witness.Ops[i].SenderPathBits[j] = bBits[j]
			witness.Ops[i].ReceiverPath[j] = aPath[j]
			witness.Ops[i].ReceiverPathBits[j] = aBits[j]
		}

		// 4. Simulate Receiver Update in Go Tree (Balance +0)
		aliceLeafDummy := HashAccountLeaf(big.NewInt(int64(aliceIndex)), bytesToBigInt(alicePub.A.X.Bytes()), bytesToBigInt(alicePub.A.Y.Bytes()), currentAliceBal, currentAliceNonce)
		tree.Update(aliceIndex, aliceLeafDummy)
	}

	witness.NewRoot = tree.Root() // Update NewRoot to reflect all ops!

	// For gnark EdDSA test on BN254
	assert.CheckCircuit(&StateTransitionCircuit{}, test.WithValidAssignment(&witness), test.WithCurves(ecc.BN254))
}
