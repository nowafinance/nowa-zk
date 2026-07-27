package circuits

import (
	"crypto/rand"
	"math/big"
	"testing"
	"os"
	"encoding/json"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	secp256k1ecdsa "github.com/consensys/gnark-crypto/ecc/secp256k1/ecdsa"
	"github.com/consensys/gnark-crypto/hash"
	"github.com/consensys/gnark/std/math/emulated"
	"github.com/consensys/gnark/test"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/backend/groth16"
	bn254 "github.com/consensys/gnark/backend/groth16/bn254"
	"github.com/consensys/gnark/backend"
	"golang.org/x/crypto/sha3"
)

func generateValidTestWitness(assert *test.Assert) *BatchTradeSignatureCircuit {
	var witness BatchTradeSignatureCircuit
	h := hash.MIMC_BN254.New()
	for i := 0; i < TradeBatchSize; i++ {
		privKey, err := secp256k1ecdsa.GenerateKey(rand.Reader)
		assert.NoError(err)
		pubKey := privKey.PublicKey

		msg := make([]byte, 32)
		_, err = rand.Read(msg)
		assert.NoError(err)

		sigBin, err := privKey.Sign(msg, nil)
		assert.NoError(err)

		var sig secp256k1ecdsa.Signature
		_, err = sig.SetBytes(sigBin)
		assert.NoError(err)

		witness.MessageHashes[i] = emulated.ValueOf[emulated.Secp256k1Fr](msg)
		xBytes := pubKey.A.X.Bytes()
		yBytes := pubKey.A.Y.Bytes()
		witness.PubKeys[i].X = emulated.ValueOf[emulated.Secp256k1Fp](xBytes[:])
		witness.PubKeys[i].Y = emulated.ValueOf[emulated.Secp256k1Fp](yBytes[:])
		rBytes := sig.R
		sBytes := sig.S
		witness.Sigs[i].R = emulated.ValueOf[emulated.Secp256k1Fr](rBytes[:])
		witness.Sigs[i].S = emulated.ValueOf[emulated.Secp256k1Fr](sBytes[:])

		msgBigInt := new(big.Int).SetBytes(msg)
		mask128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
		part1 := new(big.Int).And(msgBigInt, mask128)
		part2 := new(big.Int).Rsh(msgBigInt, 128)
		
		var part1Bytes [32]byte
		part1.FillBytes(part1Bytes[:])
		var part2Bytes [32]byte
		part2.FillBytes(part2Bytes[:])

		h.Write(part1Bytes[:])
		h.Write(part2Bytes[:])
	}
	rootBytes := h.Sum(nil)
	witness.BatchRoot = rootBytes
	return &witness
}

func TestBatchTradeSignatureCircuit_Valid(t *testing.T) {
	assert := test.NewAssert(t)
	witness := generateValidTestWitness(assert)
	assert.ProverSucceeded(&BatchTradeSignatureCircuit{}, witness, test.WithCurves(ecc.BN254), test.WithProverOpts(backend.WithProverHashToFieldFunction(sha3.NewLegacyKeccak256())), test.WithVerifierOpts(backend.WithVerifierHashToFieldFunction(sha3.NewLegacyKeccak256())))
}

func TestBatchTradeSignatureCircuit_InvalidSignature(t *testing.T) {
	assert := test.NewAssert(t)
	witness := generateValidTestWitness(assert)
	invalidMsg := make([]byte, 32)
	rand.Read(invalidMsg)
	witness.MessageHashes[0] = emulated.ValueOf[emulated.Secp256k1Fr](invalidMsg)
	assert.ProverFailed(&BatchTradeSignatureCircuit{}, witness, test.WithCurves(ecc.BN254), test.WithProverOpts(backend.WithProverHashToFieldFunction(sha3.NewLegacyKeccak256())), test.WithVerifierOpts(backend.WithVerifierHashToFieldFunction(sha3.NewLegacyKeccak256())))
}

func TestBatchTradeSignatureCircuit_InvalidBatchRoot(t *testing.T) {
	assert := test.NewAssert(t)
	witness := generateValidTestWitness(assert)
	invalidRoot := make([]byte, 32)
	rand.Read(invalidRoot)
	witness.BatchRoot = invalidRoot
	assert.ProverFailed(&BatchTradeSignatureCircuit{}, witness, test.WithCurves(ecc.BN254), test.WithProverOpts(backend.WithProverHashToFieldFunction(sha3.NewLegacyKeccak256())), test.WithVerifierOpts(backend.WithVerifierHashToFieldFunction(sha3.NewLegacyKeccak256())))
}

func TestGenerateSolidityTestData(t *testing.T) {
	assert := test.NewAssert(t)
	var circuit BatchTradeSignatureCircuit
	
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	assert.NoError(err)
	
	pk, vk, err := groth16.Setup(ccs)
	assert.NoError(err)
	
	witnessData := generateValidTestWitness(assert)
	witness, err := frontend.NewWitness(witnessData, ecc.BN254.ScalarField())
	assert.NoError(err)
	
	pubWitness, err := witness.Public()
	assert.NoError(err)
	
	proof, err := groth16.Prove(ccs, pk, witness, backend.WithProverHashToFieldFunction(sha3.NewLegacyKeccak256()))
	assert.NoError(err)
	
	err = groth16.Verify(proof, vk, pubWitness, backend.WithVerifierHashToFieldFunction(sha3.NewLegacyKeccak256()))
	assert.NoError(err)
	
	bn254Proof := proof.(*bn254.Proof)
	proof8 := [8]string{
		bn254Proof.Ar.X.String(), bn254Proof.Ar.Y.String(),
		bn254Proof.Bs.X.A1.String(), bn254Proof.Bs.X.A0.String(),
		bn254Proof.Bs.Y.A1.String(), bn254Proof.Bs.Y.A0.String(),
		bn254Proof.Krs.X.String(), bn254Proof.Krs.Y.String(),
	}

	commitments := [2]string{"0", "0"}
	if len(bn254Proof.Commitments) > 0 {
		commitments[0] = bn254Proof.Commitments[0].X.String()
		commitments[1] = bn254Proof.Commitments[0].Y.String()
	}

	commitmentPok := [2]string{"0", "0"}
	if len(bn254Proof.CommitmentPok.X) > 0 {
		commitmentPok[0] = bn254Proof.CommitmentPok.X.String()
		commitmentPok[1] = bn254Proof.CommitmentPok.Y.String()
	}

	vec := pubWitness.Vector().(fr.Vector)
	pubInputs := make([]string, len(vec))
	for i, v := range vec {
		pubInputs[i] = v.String()
	}

	data := map[string]interface{}{
		"proof": proof8,
		"commitments": commitments,
		"commitmentPok": commitmentPok,
		"publicInputs": pubInputs,
	}

	b, _ := json.MarshalIndent(data, "", "  ")
	os.MkdirAll("../../contracts/test/data", 0755)
	os.WriteFile("../../contracts/test/data/test_proof.json", b, 0644)
	
	f, err := os.Create("../../contracts/test/data/TestVerifier.sol")
	assert.NoError(err)
	defer f.Close()
	vk.ExportSolidity(f)
}
