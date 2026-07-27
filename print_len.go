package main

import (
    "fmt"
    "os"
    "bytes"
    "github.com/consensys/gnark-crypto/ecc/bn254"
    "github.com/consensys/gnark/backend/groth16"
)

func main() {
    f, _ := os.ReadFile("prover/keys/trade.pk")
    fmt.Printf("Loaded PK of size: %d\n", len(f))
}
