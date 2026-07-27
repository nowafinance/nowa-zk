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
	args := bytes[4:]
	offset := 32 + 32 + 256 + 64 + 64 + 3872
	fmt.Printf("messageHashes starts at chunk %d\n", offset/32)
	chunk := args[offset : offset+32]
	fmt.Printf("messageHashes[0]: %x\n", chunk)
}
