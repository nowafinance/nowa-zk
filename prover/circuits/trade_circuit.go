package circuits

import (
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/algebra/emulated/sw_emulated"
	"github.com/consensys/gnark/std/hash/mimc"
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

// TradeBatchSize defines the number of trades in a batch
const TradeBatchSize = 10

// BatchTradeSignatureCircuit verifies multiple ECDSA signatures
type BatchTradeSignatureCircuit struct {
	BatchRoot     frontend.Variable `gnark:",public"`
	MessageHashes [TradeBatchSize]emulated.Element[emulated.Secp256k1Fr] `gnark:",public"`
	Sigs          [TradeBatchSize]ecdsa.Signature[emulated.Secp256k1Fr]
	PubKeys       [TradeBatchSize]ecdsa.PublicKey[emulated.Secp256k1Fp, emulated.Secp256k1Fr] `gnark:",public"`
}

// Define declares the circuit constraints for batch
func (c *BatchTradeSignatureCircuit) Define(api frontend.API) error {
	params := sw_emulated.GetSecp256k1Params()
	
	f, err := emulated.NewField[emulated.Secp256k1Fr](api)
	if err != nil {
		return err
	}
	h, err := mimc.NewMiMC(api)
	if err != nil {
		return err
	}

	for i := 0; i < TradeBatchSize; i++ {
		// Verify signature
		c.PubKeys[i].Verify(api, params, &c.MessageHashes[i], &c.Sigs[i])
		
		// Hash the message hash for BatchRoot
		bits := f.ToBitsCanonical(&c.MessageHashes[i])
		// Pack 256 bits into two BN254 variables
		part1 := api.FromBinary(bits[:128]...)
		part2 := api.FromBinary(bits[128:]...)
		h.Write(part1)
		h.Write(part2)
	}
	
	// Enforce BatchRoot
	computedRoot := h.Sum()
	api.AssertIsEqual(computedRoot, c.BatchRoot)
	
	return nil
}
