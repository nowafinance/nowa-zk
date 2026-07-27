package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

func main() {
	data, err := os.ReadFile("tx_input.hex")
	if err != nil {
		panic(err)
	}
	hexStr := strings.TrimSpace(string(data))
	if strings.HasPrefix(hexStr, "0x") {
		hexStr = hexStr[2:]
	}
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		panic(err)
	}
	// The function selector is 4 bytes
	if len(bytes) < 4 {
		panic("too short")
	}
	fmt.Printf("Selector: %x\n", bytes[:4])
	args := bytes[4:]
	// args is a sequence of 32-byte chunks
	fmt.Printf("Number of 32-byte chunks: %d\n", len(args)/32)
	// Let's find publicInputs in the ABI encoding
	// The registerTrades arguments are:
	// batchNumber (uint256) -> 32 bytes
	// chunkIndex (uint256) -> 32 bytes
	// proof (uint256[8]) -> 256 bytes
	// commitments (uint256[2]) -> 64 bytes
	// commitmentPok (uint256[2]) -> 64 bytes
	// publicInputs (uint256[121]) -> 3872 bytes
	// ...
	offset := 32 + 32 + 256 + 64 + 64
	fmt.Printf("publicInputs starts at chunk %d\n", offset/32)
	for i := 0; i < 5; i++ {
		chunk := args[offset+i*32 : offset+(i+1)*32]
		fmt.Printf("publicInputs[%d]: %x\n", i, chunk)
	}
}
