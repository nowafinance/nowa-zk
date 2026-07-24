package main

import (
	"encoding/json"
	"fmt"
	"math/big"
)

type DummyUnpacked struct {
	Nonce *big.Int
}

type Order struct {
	Nonce *big.Int
}

func main() {
	input := DummyUnpacked{
		Nonce: big.NewInt(1000),
	}
	
	b, err := json.Marshal(input)
	if err != nil {
		fmt.Printf("Marshal error: %v\n", err)
		return
	}
	fmt.Printf("Marshalled: %s\n", string(b))
	
	var output Order
	if err := json.Unmarshal(b, &output); err != nil {
		fmt.Printf("Unmarshal error: %v\n", err)
		return
	}
	fmt.Printf("Unmarshalled: %v\n", output.Nonce)
}
