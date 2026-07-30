package circuits

import (
	"github.com/consensys/gnark-crypto/ecc/twistededwards"
	"github.com/consensys/gnark/frontend"
	gnark_twistededwards "github.com/consensys/gnark/std/algebra/native/twistededwards"
	"github.com/consensys/gnark/std/hash/mimc"
	"github.com/consensys/gnark/std/signature/eddsa"
)

// Operation type tags.
const (
	OpTrade      = 0
	OpTransfer   = 1
	OpWithdrawal = 2
)

// MerkleDepth sizes the account tree to 2^MerkleDepth leaves.
const MerkleDepth = 20

// BatchSize for the State Transition Circuit
const BatchSize = 5

// Operation represents one state-mutating action inside a batch.
type Operation struct {
	OpType frontend.Variable

	MessageHash frontend.Variable // Native BN254 field element
	Sig         eddsa.Signature   // BabyJubJub Signature
	PubKey      eddsa.PublicKey   // BabyJubJub Public Key

	SenderIndex    frontend.Variable
	SenderBalance  frontend.Variable
	SenderNonce    frontend.Variable
	SenderPath     [MerkleDepth]frontend.Variable
	SenderPathBits [MerkleDepth]frontend.Variable

	// Receiver public key (no signature needed)
	ReceiverPubKey   eddsa.PublicKey
	ReceiverIndex    frontend.Variable
	ReceiverBalance  frontend.Variable
	ReceiverNonce    frontend.Variable
	ReceiverPath     [MerkleDepth]frontend.Variable
	ReceiverPathBits [MerkleDepth]frontend.Variable

	Amount frontend.Variable
}

// StateTransitionCircuit proves that applying BatchSize signed, authorized
// operations to OldRoot correctly and losslessly produces NewRoot.
// It also accumulates all withdrawal amounts into WithdrawalHash.
type StateTransitionCircuit struct {
	OldRoot        frontend.Variable `gnark:",public"`
	NewRoot        frontend.Variable `gnark:",public"`
	WithdrawalHash frontend.Variable `gnark:",public"`

	Ops [BatchSize]Operation
}

// accountLeaf computes the MiMC hash of an account's state.
func accountLeaf(h *mimc.MiMC, index, pubX, pubY, balance, nonce frontend.Variable) frontend.Variable {
	h.Reset()
	h.Write(index, pubX, pubY, balance, nonce)
	return h.Sum()
}

// merkleRoot recomputes a root from a leaf + authentication path.
// bits[i] == 0 means LEFT child, bits[i] == 1 means RIGHT child.
func merkleRoot(h *mimc.MiMC, api frontend.API, leaf frontend.Variable, path, bits [MerkleDepth]frontend.Variable) frontend.Variable {
	cur := leaf
	for i := 0; i < MerkleDepth; i++ {
		left := api.Select(bits[i], path[i], cur)
		right := api.Select(bits[i], cur, path[i])
		h.Reset()
		h.Write(left, right)
		cur = h.Sum()
	}
	return cur
}

// rangeCheck prevents field-modulus wraparound by ensuring the variable fits in 252 bits.
func rangeCheck(api frontend.API, val frontend.Variable) {
	api.ToBinary(val, 252)
}

func (c *StateTransitionCircuit) Define(api frontend.API) error {
	// 1. Initialize the Twisted Edwards Curve (BabyJubJub)
	curve, err := gnark_twistededwards.NewEdCurve(api, twistededwards.BN254)
	if err != nil {
		return err
	}

	h, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}

	root := c.OldRoot
	
	// Start with a zeroed withdrawal accumulator
	currentWithdrawalHash := frontend.Variable(0)

	for i := 0; i < BatchSize; i++ {
		op := &c.Ops[i]

		// Ensure Amount fits in 252 bits (prevents malicious underflow/wraparound)
		rangeCheck(api, op.Amount)

		// 2. Verify BabyJubJub Signature (This is 100x cheaper than ECDSA!)
		h.Reset() // Ensure hash is completely clean before EdDSA uses it!
		err = eddsa.Verify(curve, op.Sig, op.MessageHash, op.PubKey, &h)
		if err != nil {
			return err
		}

		isWithdrawal := api.IsZero(api.Sub(op.OpType, OpWithdrawal))

		// For withdrawals, Receiver MUST be the Sender (no-op credit).
		enforcedReceiverIndex := api.Select(isWithdrawal, op.SenderIndex, op.ReceiverIndex)
		api.AssertIsEqual(enforcedReceiverIndex, api.Select(isWithdrawal, op.ReceiverIndex, op.ReceiverIndex))

		// --- Withdrawal Accumulator ---
		// If OpType == Withdrawal, hash(currentHash, senderIndex, amount)
		h.Reset()
		h.Write(currentWithdrawalHash, op.SenderIndex, op.Amount)
		nextWithdrawalHash := h.Sum()
		// Only update accumulator if it actually is a withdrawal
		currentWithdrawalHash = api.Select(isWithdrawal, nextWithdrawalHash, currentWithdrawalHash)

		// --- Debit sender ---
		oldSenderLeaf := accountLeaf(&h, op.SenderIndex, op.PubKey.A.X, op.PubKey.A.Y, op.SenderBalance, op.SenderNonce)
		computedRoot := merkleRoot(&h, api, oldSenderLeaf, op.SenderPath, op.SenderPathBits)
		api.AssertIsEqual(computedRoot, root)

		// Ensure sender has enough balance
		api.AssertIsLessOrEqual(op.Amount, op.SenderBalance)
		newSenderBalance := api.Sub(op.SenderBalance, op.Amount)
		newSenderNonce := api.Add(op.SenderNonce, 1)
		
		newSenderLeaf := accountLeaf(&h, op.SenderIndex, op.PubKey.A.X, op.PubKey.A.Y, newSenderBalance, newSenderNonce)
		root = merkleRoot(&h, api, newSenderLeaf, op.SenderPath, op.SenderPathBits)

		// --- Credit receiver ---
		// creditAmount is zero for withdrawals.
		creditAmount := api.Select(isWithdrawal, 0, op.Amount)
		
		oldReceiverLeaf := accountLeaf(&h, op.ReceiverIndex, op.ReceiverPubKey.A.X, op.ReceiverPubKey.A.Y, op.ReceiverBalance, op.ReceiverNonce)
		computedRoot2 := merkleRoot(&h, api, oldReceiverLeaf, op.ReceiverPath, op.ReceiverPathBits)
		api.AssertIsEqual(computedRoot2, root)

		newReceiverBalance := api.Add(op.ReceiverBalance, creditAmount)
		// Receiver nonce does not increment, they didn't sign anything.
		newReceiverLeaf := accountLeaf(&h, op.ReceiverIndex, op.ReceiverPubKey.A.X, op.ReceiverPubKey.A.Y, newReceiverBalance, op.ReceiverNonce)
		root = merkleRoot(&h, api, newReceiverLeaf, op.ReceiverPath, op.ReceiverPathBits)
	}

	// Final validations
	api.AssertIsEqual(root, c.NewRoot)
	api.AssertIsEqual(currentWithdrawalHash, c.WithdrawalHash)

	return nil
}
