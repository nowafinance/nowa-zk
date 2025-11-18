package prover

import (
	"fmt"

	"github.com/tannetwork/zk-sequencer/prover/internal/witness"
)

// RunProve orchestrates a proof run (placeholder).
// Replace with actual gnark compile/setup/prove/verify pipeline.
func RunProve(circuitID string) error {
	w, err := witness.BuildExample()
	if err != nil {
		return fmt.Errorf("build witness: %w", err)
	}
	_ = w

	// TODO: compile circuits (frontend.Compile), run groth16.Setup, groth16.Prove, etc.
	// Example flow:
	//  - import circuit from circuits/ and compile with frontend.Compile
	//  - create witness with witness package
	//  - run backend/groth16 Setup/Prove/Verify

	return nil
}
