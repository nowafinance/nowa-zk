package main

import (
    "fmt"
    "os"
)

func main() {
    f, _ := os.ReadFile("keys/trade.pk")
    fmt.Printf("Loaded PK of size: %d\n", len(f))
}
