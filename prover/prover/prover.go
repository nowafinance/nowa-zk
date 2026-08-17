package prover

// RunProve is a legacy stub — the real proving pipeline lives in:
//   cmd/prover/start.go → start() → generateProof() → verifyLocal() → submitProof()
//
// To run the prover:
//   ./prover-bin start --keys-dir ./keys --contract <addr> --private-key <key>
func RunProve(circuitID string) error {
	return nil
}
