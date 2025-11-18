package prover

import (
	"testing"

	// blank-import gnark packages to ensure they compile when running `go test`
	_ "github.com/consensys/gnark/frontend"
	_ "github.com/consensys/gnark/backend/groth16"
)

func TestCompileGnarkDeps(t *testing.T) {
	// Compiles gnark deps via blank imports above.
}
