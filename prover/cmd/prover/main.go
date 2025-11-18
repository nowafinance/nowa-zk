package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tannetwork/zk-sequencer/prover/prover"
)

func main() {
	circuit := flag.String("circuit", "circuits/simple", "circuit identifier or path")
	flag.Parse()

	if err := prover.RunProve(*circuit); err != nil {
		fmt.Fprintln(os.Stderr, "prove failed:", err)
		os.Exit(1)
	}
	fmt.Println("prove completed")
}
