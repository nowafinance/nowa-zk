package main

import (
    "fmt"

    "github.com/tannetwork/zk-sequencer/prover/internal/prover"
)

func main() {
    fmt.Println("zk-prover: starting up…")
    p := prover.New()
    fmt.Println(p.Info())
}


