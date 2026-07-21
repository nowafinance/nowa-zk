package circuits

import (
	"crypto/rand"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	secp256k1ecdsa "github.com/consensys/gnark-crypto/ecc/secp256k1/ecdsa"
	"github.com/consensys/gnark/std/math/emulated"
	"github.com/consensys/gnark/test"
)

func TestTradeSignatureCircuit(t *testing.T) {
	assert := test.NewAssert(t)

	// 1. Generate secp256k1 keypair
	privKey, err := secp256k1ecdsa.GenerateKey(rand.Reader)
	assert.NoError(err)
	pubKey := privKey.PublicKey

	// 2. Generate a random MessageHash
	msg := make([]byte, 32)
	_, err = rand.Read(msg)
	assert.NoError(err)

	// 3. Sign the message
	sigBin, err := privKey.Sign(msg, nil)
	assert.NoError(err)

	var sig secp256k1ecdsa.Signature
	_, err = sig.SetBytes(sigBin)
	assert.NoError(err)

	// 4. Assign values to circuit witness
	var witness TradeSignatureCircuit

	witness.MessageHash = emulated.ValueOf[emulated.Secp256k1Fr](msg)

	// In gnark-crypto, A.X and A.Y are of type fp.Element.
	// We convert them to bytes first to ensure emulated.ValueOf can parse them.
	xBytes := pubKey.A.X.Bytes()
	yBytes := pubKey.A.Y.Bytes()
	witness.PubKey.X = emulated.ValueOf[emulated.Secp256k1Fp](xBytes[:])
	witness.PubKey.Y = emulated.ValueOf[emulated.Secp256k1Fp](yBytes[:])

	rBytes := sig.R
	sBytes := sig.S
	witness.Sig.R = emulated.ValueOf[emulated.Secp256k1Fr](rBytes[:])
	witness.Sig.S = emulated.ValueOf[emulated.Secp256k1Fr](sBytes[:])

	// 5. Test circuit
	// We test with BN254 which is standard for Ethereum rollups.
	assert.ProverSucceeded(&TradeSignatureCircuit{}, &witness, test.WithCurves(ecc.BN254))
}
