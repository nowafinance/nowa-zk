package circuits

import (
	"crypto/rand"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/consensys/gnark-crypto/ecc/twistededwards"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// buildFullBatchWitness constructs a real BatchSize-op witness — BatchSize sequential
// transfers between the same two accounts, mirroring exactly how a real Sequencer
// batch accumulates fills. Deliberately mirrors
// TestStateTransitionCircuit_FullBatchOf25's construction (kept separate rather than
// factored out of that test, so BENCHMARKS.md numbers below come from the same
// real-world-shaped witness every full-batch proof actually uses, without coupling
// this file to that test's internals).
func buildFullBatchWitness(tb testing.TB) *StateTransitionCircuit {
	tb.Helper()
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
	senderBalance := int64(BatchSize * transferAmount)
	senderNonce := int64(0)
	receiverBalance := int64(0)

	smt.Update(0, hashGo(big.NewInt(0), senderX, senderY, big.NewInt(senderBalance), big.NewInt(senderNonce)))
	smt.Update(1, hashGo(big.NewInt(1), receiverX, receiverY, big.NewInt(receiverBalance), big.NewInt(0)))

	witness := &StateTransitionCircuit{}
	witness.OldRoot = smt.Root()
	witness.WithdrawalHash = 0
	witness.DepositHash = 0

	emptyPath, emptyBits := smt.GetPath(99)

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

		witness.Ops[i].TakerPubKey.Assign(twistededwards.BN254, receiverPub.Bytes())
		witness.Ops[i].TakerSig.R.X = 0
		witness.Ops[i].TakerSig.R.Y = 0
		witness.Ops[i].TakerSig.S = 0

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
	return witness
}

// loadOrSetupKeys loads the real proving/verifying keys and compiled circuit that
// `make setup` already generated at ~/.nowa-zk/keys/state.{ccs,pk,vk} — the exact
// artifacts the live Prover uses (see prover/cmd/prover/start.go's
// loadCircuitAndKeys) — rather than re-running groth16.Setup, which allocates
// several GB for the SRS/MSM step on top of whatever else is running on the
// machine. Falls back to a fresh Compile+Setup only if those files aren't present.
func loadOrSetupKeys(b *testing.B) (constraint.ConstraintSystem, groth16.ProvingKey, groth16.VerifyingKey) {
	b.Helper()
	home, err := os.UserHomeDir()
	if err == nil {
		ccsPath := filepath.Join(home, ".nowa-zk", "keys", "state.ccs")
		pkPath := filepath.Join(home, ".nowa-zk", "keys", "state.pk")
		vkPath := filepath.Join(home, ".nowa-zk", "keys", "state.vk")
		if ccsFile, err := os.Open(ccsPath); err == nil {
			defer ccsFile.Close()
			ccs := groth16.NewCS(ecc.BN254)
			readStart := time.Now()
			if _, err := ccs.ReadFrom(ccsFile); err != nil {
				b.Fatalf("read %s: %v", ccsPath, err)
			}
			pkFile, err := os.Open(pkPath)
			if err != nil {
				b.Fatalf("open %s: %v", pkPath, err)
			}
			defer pkFile.Close()
			pk := groth16.NewProvingKey(ecc.BN254)
			if _, err := pk.ReadFrom(pkFile); err != nil {
				b.Fatalf("read %s: %v", pkPath, err)
			}
			vkFile, err := os.Open(vkPath)
			if err != nil {
				b.Fatalf("open %s: %v", vkPath, err)
			}
			defer vkFile.Close()
			vk := groth16.NewVerifyingKey(ecc.BN254)
			if _, err := vk.ReadFrom(vkFile); err != nil {
				b.Fatalf("read %s: %v", vkPath, err)
			}
			b.Logf("Loaded real keys from ~/.nowa-zk/keys in %s (%d constraints, BatchSize=%d)", time.Since(readStart), ccs.GetNbConstraints(), BatchSize)
			return ccs, pk, vk
		}
	}

	b.Log("no ~/.nowa-zk/keys found — compiling + running groth16.Setup fresh")
	circuit := &StateTransitionCircuit{}
	compileStart := time.Now()
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("Compile: %s (%d constraints, BatchSize=%d)", time.Since(compileStart), ccs.GetNbConstraints(), BatchSize)
	setupStart := time.Now()
	pk, vk, err := groth16.Setup(ccs)
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("Setup: %s", time.Since(setupStart))
	return ccs, pk, vk
}

// BenchmarkStateTransitionCircuit times Prove/Verify — the actual per-batch cost —
// against the real, currently-deployed proving/verifying keys, using a real full
// BatchSize-op witness with genuine EdDSA signatures.
//
// Run with: cd prover && go test -bench=BenchmarkStateTransitionCircuit -benchtime=1x -run=^$ -v ./circuits/
func BenchmarkStateTransitionCircuit(b *testing.B) {
	witness := buildFullBatchWitness(b)
	ccs, pk, vk := loadOrSetupKeys(b)

	witnessStart := time.Now()
	fullWitness, err := frontend.NewWitness(witness, ecc.BN254.ScalarField())
	if err != nil {
		b.Fatal(err)
	}
	publicWitness, err := fullWitness.Public()
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("Witness generation: %s", time.Since(witnessStart))

	var proof groth16.Proof
	b.Run("Prove", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			proof, err = groth16.Prove(ccs, pk, fullWitness)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Verify", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := groth16.Verify(proof, vk, publicWitness); err != nil {
				b.Fatal(err)
			}
		}
	})
}
