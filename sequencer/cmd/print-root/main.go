// print-root opens the Sequencer's Merkle tree (empty or not — an empty tree's
// root is not literally 0, it's the SMT's zero-node convention hashed up 28 levels)
// and prints its current root as a decimal string on stdout, nothing else.
//
// Exists to automate what used to be a manual, easy-to-forget deploy step: a fresh
// NowaRollup starts at whatever INITIAL_STATE_ROOT it's given, and no real Sequencer
// tree (empty or not) ever actually roots to the old hardcoded default of 0 — see
// docs/operations/troubleshooting.md's history of that exact revert. `make deploy`
// now calls this first and passes its output straight through as INITIAL_STATE_ROOT,
// so the contract starts in sync with whatever the Sequencer already has, and the
// separate post-deploy setStateRoot() bootstrap step is no longer needed for a fresh
// deploy (still needed if you deploy a new contract against an already-batching
// Sequencer, in which case pass INITIAL_STATE_ROOT explicitly instead).
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
)

func main() {
	dataDir := flag.String("data-dir", "./nowa_state_db", "Sequencer LevelDB data directory")
	flag.Parse()

	tree, err := state.NewLevelDBMerkleTree(*dataDir, 28)
	if err != nil {
		fmt.Fprintf(os.Stderr, "print-root: open %s: %v\n", *dataDir, err)
		os.Exit(1)
	}
	defer tree.Close()

	// forge's vm.envBytes32 expects 0x-prefixed, zero-padded hex — not the decimal
	// string the rest of this codebase otherwise uses for roots (e.g. in batch JSON).
	root := tree.Root()
	rootBytes := root.FillBytes(make([]byte, 32))
	fmt.Printf("0x%x\n", rootBytes)
}
