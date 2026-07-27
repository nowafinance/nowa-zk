package circuits

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	secp256k1ecdsa "github.com/consensys/gnark-crypto/ecc/secp256k1/ecdsa"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	"github.com/consensys/gnark/std/math/emulated"
	gomimc "github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"math/big"
)

// generateBatchTradeWitness builds a fully populated BatchTradeSignatureCircuit witness
// using freshly generated secp256k1 keypairs and random message hashes.
func generateBatchTradeWitness() BatchTradeSignatureCircuit {
	var witness BatchTradeSignatureCircuit
	var msgs [TradeBatchSize][]byte
	for i := 0; i < TradeBatchSize; i++ {
		privKey, err := secp256k1ecdsa.GenerateKey(rand.Reader)
		if err != nil {
			panic(err)
		}
		msg := make([]byte, 32)
		if _, err := rand.Read(msg); err != nil {
			panic(err)
		}
		msgs[i] = msg
		sigBin, err := privKey.Sign(msg, nil)
		if err != nil {
			panic(err)
		}
		var sig secp256k1ecdsa.Signature
		if _, err := sig.SetBytes(sigBin); err != nil {
			panic(err)
		}

		witness.MessageHashes[i] = emulated.ValueOf[emulated.Secp256k1Fr](msg)
		xBytes := privKey.PublicKey.A.X.Bytes()
		yBytes := privKey.PublicKey.A.Y.Bytes()
		witness.PubKeys[i].X = emulated.ValueOf[emulated.Secp256k1Fp](xBytes[:])
		witness.PubKeys[i].Y = emulated.ValueOf[emulated.Secp256k1Fp](yBytes[:])
		witness.Sigs[i].R = emulated.ValueOf[emulated.Secp256k1Fr](sig.R[:])
		witness.Sigs[i].S = emulated.ValueOf[emulated.Secp256k1Fr](sig.S[:])
	}
	
	// Compute BatchRoot
	goMiMC := gomimc.NewMiMC()
	for i := 0; i < TradeBatchSize; i++ {
		val := new(big.Int).SetBytes(msgs[i])
		
		mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
		part1 := new(big.Int).And(val, mask)
		part2 := new(big.Int).Rsh(val, 128)
		
		b1 := make([]byte, 32)
		part1.FillBytes(b1)
		goMiMC.Write(b1)
		
		b2 := make([]byte, 32)
		part2.FillBytes(b2)
		goMiMC.Write(b2)
	}
	witness.BatchRoot = goMiMC.Sum(nil)
	return witness
}

// BenchmarkCircuitSetup measures the time to compile the BatchTradeSignatureCircuit
// and run the Groth16 trusted setup.
func BenchmarkCircuitSetup(b *testing.B) {
	emptyCircuit := BatchTradeSignatureCircuit{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &emptyCircuit)
		if err != nil {
			b.Fatal(err)
		}
		_, _, err = groth16.Setup(ccs)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("Setup Time: %s", time.Since(start))
	}
}

// BenchmarkWitnessGeneration measures the time to generate the witness from trade inputs.
func BenchmarkWitnessGeneration(b *testing.B) {
	witness := generateBatchTradeWitness()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := frontend.NewWitness(&witness, ecc.BN254.ScalarField())
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("Witness Generation Time: %s", time.Since(start))
	}
}

// BenchmarkProofGeneration measures ONLY the Groth16 proof generation time.
func BenchmarkProofGeneration(b *testing.B) {
	witness := generateBatchTradeWitness()

	emptyCircuit := BatchTradeSignatureCircuit{}
	ccs, _ := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &emptyCircuit)
	pk, _, _ := groth16.Setup(ccs)
	w, _ := frontend.NewWitness(&witness, ecc.BN254.ScalarField())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, err := groth16.Prove(ccs, pk, w)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("Proof Generation Time: %s", time.Since(start))
	}
}

// BenchmarkVerification measures the time to verify a Groth16 proof.
func BenchmarkVerification(b *testing.B) {
	witness := generateBatchTradeWitness()

	emptyCircuit := BatchTradeSignatureCircuit{}
	ccs, _ := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &emptyCircuit)
	pk, vk, _ := groth16.Setup(ccs)
	w, _ := frontend.NewWitness(&witness, ecc.BN254.ScalarField())
	proof, _ := groth16.Prove(ccs, pk, w)
	publicWitness, _ := w.Public()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start := time.Now()
		err := groth16.Verify(proof, vk, publicWitness)
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("Verification Time: %s", time.Since(start))
	}
}
