package circuits

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	secp256k1ecdsa "github.com/consensys/gnark-crypto/ecc/secp256k1/ecdsa"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/std/math/emulated"
	"github.com/consensys/gnark/std/signature/ecdsa"
	"github.com/consensys/gnark/test"
)

func TestStateTransitionCircuitCompilation(t *testing.T) {
	assert := test.NewAssert(t)
	var circuit StateTransitionCircuit

	// Just checking if the circuit compiles and is valid structurally.
	_, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	assert.NoError(err)
}

func TestValidTransfer(t *testing.T) {
	assert := test.NewAssert(t)

	// 1. Initialize our Go Merkle Tree
	tree := NewSparseMerkleTree(MerkleDepth)

	// 2. Create Alice (Sender)
	alicePriv, _ := secp256k1ecdsa.GenerateKey(rand.Reader)
	alicePub := alicePriv.PublicKey
	aliceIndex := uint64(1)
	aliceBal := big.NewInt(1000)
	aliceNonce := big.NewInt(0)
	
	aliceLeaf := HashAccountLeaf(big.NewInt(int64(aliceIndex)), alicePub.A.X.BigInt(new(big.Int)), alicePub.A.Y.BigInt(new(big.Int)), aliceBal, aliceNonce)
	tree.Update(aliceIndex, aliceLeaf)

	// 3. Create Bob (Receiver)
	bobPriv, _ := secp256k1ecdsa.GenerateKey(rand.Reader)
	bobPub := bobPriv.PublicKey
	bobIndex := uint64(2)
	bobBal := big.NewInt(500)
	bobNonce := big.NewInt(0)
	
	bobLeaf := HashAccountLeaf(big.NewInt(int64(bobIndex)), bobPub.A.X.BigInt(new(big.Int)), bobPub.A.Y.BigInt(new(big.Int)), bobBal, bobNonce)
	tree.Update(bobIndex, bobLeaf)

	oldRoot := tree.Root()

	// 4. Create the Transfer (Alice sends 100 to Bob)
	amount := big.NewInt(100)
	msg := make([]byte, 32)
	rand.Read(msg) // Dummy message hash for the signature
	msg[0] = 0 // Ensure < curve order
	sigBin, _ := alicePriv.Sign(msg, nil)
	var sig secp256k1ecdsa.Signature
	sig.SetBytes(sigBin)

	// Get Merkle proofs for Alice
	alicePath, aliceBits := tree.GetPath(aliceIndex)

	// Update Alice's state in the tree
	newAliceBal := new(big.Int).Sub(aliceBal, amount)
	newAliceNonce := new(big.Int).Add(aliceNonce, big.NewInt(1))
	newAliceLeaf := HashAccountLeaf(big.NewInt(int64(aliceIndex)), alicePub.A.X.BigInt(new(big.Int)), alicePub.A.Y.BigInt(new(big.Int)), newAliceBal, newAliceNonce)
	tree.Update(aliceIndex, newAliceLeaf)

	// Get Merkle proofs for Bob (Using the tree AFTER Alice's update)
	bobPath, bobBits := tree.GetPath(bobIndex)

	// Update Bob's state in the tree
	newBobBal := new(big.Int).Add(bobBal, amount)
	newBobLeaf := HashAccountLeaf(big.NewInt(int64(bobIndex)), bobPub.A.X.BigInt(new(big.Int)), bobPub.A.Y.BigInt(new(big.Int)), newBobBal, bobNonce)
	tree.Update(bobIndex, newBobLeaf)

	newRoot := tree.Root()

	// 5. Construct the Witness
	var witness StateTransitionCircuit
	witness.OldRoot = oldRoot
	witness.NewRoot = newRoot
	witness.WithdrawalHash = 0 // No withdrawals in this batch

	// Fill the first operation
	witness.Ops[0] = Operation{
		OpType:        OpTransfer,
		MessageHash:   emulated.ValueOf[emulated.Secp256k1Fr](msg),
		Sig:           gnarkSig(sig),
		PubKey:        gnarkPubKey(alicePub),
		SenderIndex:   aliceIndex,
		SenderBalance: aliceBal,
		SenderNonce:   aliceNonce,
		Amount:        amount,
		
		ReceiverPubKey:  gnarkPubKey(bobPub),
		ReceiverIndex:   bobIndex,
		ReceiverBalance: bobBal,
		ReceiverNonce:   bobNonce,
	}

	for i := 0; i < MerkleDepth; i++ {
		witness.Ops[0].SenderPath[i] = alicePath[i]
		witness.Ops[0].SenderPathBits[i] = aliceBits[i]
		witness.Ops[0].ReceiverPath[i] = bobPath[i]
		witness.Ops[0].ReceiverPathBits[i] = bobBits[i]
	}

	// Fill the rest with dummy NO-OPs (Transfer 0 from Bob to Bob to pass validation)
	for i := 1; i < BatchSize; i++ {
		msgDummy := make([]byte, 32)
		rand.Read(msgDummy)
		msgDummy[0] = 0
		sigBinDummy, _ := bobPriv.Sign(msgDummy, nil)
		var sigDummy secp256k1ecdsa.Signature
		sigDummy.SetBytes(sigBinDummy)

		bPath, bBits := tree.GetPath(bobIndex)
		witness.Ops[i] = Operation{
			OpType:          OpTransfer,
			MessageHash:     emulated.ValueOf[emulated.Secp256k1Fr](msgDummy),
			Sig:             gnarkSig(sigDummy),
			PubKey:          gnarkPubKey(bobPub),
			SenderIndex:     bobIndex,
			SenderBalance:   newBobBal,
			SenderNonce:     bobNonce,
			Amount:          0,
			ReceiverPubKey:  gnarkPubKey(bobPub),
			ReceiverIndex:   bobIndex,
			ReceiverBalance: newBobBal,
			ReceiverNonce:   bobNonce,
		}
		for j := 0; j < MerkleDepth; j++ {
			witness.Ops[i].SenderPath[j] = bPath[j]
			witness.Ops[i].SenderPathBits[j] = bBits[j]
			witness.Ops[i].ReceiverPath[j] = bPath[j]
			witness.Ops[i].ReceiverPathBits[j] = bBits[j]
		}
	}

	assert.CheckCircuit(&StateTransitionCircuit{}, test.WithValidAssignment(&witness))
}

func gnarkSig(sig secp256k1ecdsa.Signature) ecdsa.Signature[emulated.Secp256k1Fr] {
	return ecdsa.Signature[emulated.Secp256k1Fr]{
		R: emulated.ValueOf[emulated.Secp256k1Fr](sig.R[:]),
		S: emulated.ValueOf[emulated.Secp256k1Fr](sig.S[:]),
	}
}

func gnarkPubKey(pub secp256k1ecdsa.PublicKey) ecdsa.PublicKey[emulated.Secp256k1Fp, emulated.Secp256k1Fr] {
	xBytes := pub.A.X.Bytes()
	yBytes := pub.A.Y.Bytes()
	return ecdsa.PublicKey[emulated.Secp256k1Fp, emulated.Secp256k1Fr]{
		X: emulated.ValueOf[emulated.Secp256k1Fp](xBytes[:]),
		Y: emulated.ValueOf[emulated.Secp256k1Fp](yBytes[:]),
	}
}
