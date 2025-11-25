package prover

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"

	"github.com/stretchr/testify/require"
	"github.com/tannetwork/zk-sequencer/prover/circuits"
)

func TestSimpleCircuitProof(t *testing.T) {
	// 1. Define the circuit
	circuit := circuits.SimpleCircuit{}

	// 2. Compile the circuit into a set of constraints
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &circuit)
	require.NoError(t, err, "Failed to compile circuit")

	// 3. Generate the proving and verification keys
	pk, vk, err := groth16.Setup(ccs)
	require.NoError(t, err, "Failed to run trusted setup")

	// 4. Define the witness (valid assignment)
	// Public input is A=2, Private input is B=3. Constraint A+B==5 is met.
	assignment := circuits.SimpleCircuit{
		A: frontend.Variable(2),
		B: frontend.Variable(3),
	}
	witness, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField())
	require.NoError(t, err, "Failed to create witness")

	// 5. Generate the proof
	proof, err := groth16.Prove(ccs, pk, witness)
	require.NoError(t, err, "Failed to generate proof")

	// 6. Verify the proof
	publicWitness, err := witness.Public()
	require.NoError(t, err, "Failed to create public witness")

	err = groth16.Verify(proof, vk, publicWitness)
	require.NoError(t, err, "Proof verification failed")
}
