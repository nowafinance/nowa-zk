package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
)

// VerifyEdDSASignature takes an order payload, a 64-byte EdDSA signature (hex string),
// and a public key (hex string of 32 bytes) and verifies it.
// The public key must match the registered public key for the account in the sequencer.
func VerifyEdDSASignature(pubKeyHex string, tokenId uint32, amount, price *big.Int, isBuy bool, nonce uint64, sigHex string) (bool, error) {
	// 1. Decode Public Key
	cleanPubKeyHex := strings.TrimPrefix(pubKeyHex, "0x")
	pubKeyBytes, err := hex.DecodeString(cleanPubKeyHex)
	if err != nil {
		return false, fmt.Errorf("invalid pubkey hex: %v", err)
	}

	var pubKey eddsa.PublicKey
	_, err = pubKey.SetBytes(pubKeyBytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse eddsa pubkey: %v", err)
	}

	// 2. Decode Signature
	sigHex = strings.TrimPrefix(sigHex, "0x")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return false, fmt.Errorf("invalid signature hex: %v", err)
	}
	if len(sigBytes) != 64 {
		return false, fmt.Errorf("invalid signature length, expected 64 bytes")
	}

	// 3. Construct Message to hash
	// We hash the order fields in a deterministic way.
	// For simplicity in the sequencer matching engine, we use SHA256 of the concatenated fields,
	// because MiMC is computationally heavy in JS. The JS frontend does SHA256(fields) -> signs it.
	// (Note: In the zk-circuit, we use MiMC. This is the L2 matching engine signature which is NOT verified inside the circuit.
	// The circuit only verifies the L2 State Update signatures. The Order matching signatures are only verified here in the Sequencer to ensure trade intent).
	
	msgStr := fmt.Sprintf("%s:%d:%s:%s:%t:%d", pubKeyHex, tokenId, amount.String(), price.String(), isBuy, nonce)
	msgHash := sha256.Sum256([]byte(msgStr))
	msgHash[0] = 0 // ensure < modulus

	// In EdDSA, the hash function used internally by the Verify function must match the one used during signing.
	// We use MiMC for the internal EdDSA scalar derivations as per Gnark standard.
	hFunc := mimc.NewMiMC()
	
	valid, err := pubKey.Verify(sigBytes, msgHash[:], hFunc)
	if err != nil {
		return false, err
	}
	
	return valid, nil
}
