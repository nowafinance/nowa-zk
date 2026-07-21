package circuits

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/algebra/emulated/sw_emulated"
	"github.com/consensys/gnark/std/math/emulated"
	"github.com/consensys/gnark/std/signature/ecdsa"
)

// TradeSignatureCircuit verifies an ECDSA signature over secp256k1
// for a given EIP-712 MessageHash.
type TradeSignatureCircuit struct {
	// MessageHash is the EIP-712 hash of the order (public input)
	MessageHash emulated.Element[emulated.Secp256k1Fr] `gnark:",public"`

	// Signature (private input, usually)
	Sig ecdsa.Signature[emulated.Secp256k1Fr]

	// Public Key of the signer
	// In a full rollup, this would be private and its Keccak256 hash
	// would be verified against the userAddress in the Order struct.
	PubKey ecdsa.PublicKey[emulated.Secp256k1Fp, emulated.Secp256k1Fr] `gnark:",public"`
}

// Define declares the circuit constraints
func (c *TradeSignatureCircuit) Define(api frontend.API) error {
	// Verify the secp256k1 ECDSA signature
	c.PubKey.Verify(api, sw_emulated.GetSecp256k1Params(), &c.MessageHash, &c.Sig)
	return nil
}
