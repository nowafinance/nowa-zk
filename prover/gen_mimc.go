package main

import (
	"fmt"
	"math/big"
	"golang.org/x/crypto/sha3"
)

func main() {
	fmt.Println(`// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

library MiMC {
    uint256 constant Q = 21888242871839275222246405745257275088548364400416034343698204186575808495617;
    
    function getConstants() internal pure returns (uint256[110] memory) {
        uint256[110] memory roundConstants;`)

	seed := []byte("seed")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(seed)
	rnd := hash.Sum(nil)
	hash.Reset()
	hash.Write(rnd)

	for i := 0; i < 110; i++ {
		rnd = hash.Sum(nil)
		val := new(big.Int).SetBytes(rnd)
		// Modulo Q just in case, like setBytesCanonical
		q, _ := new(big.Int).SetString("21888242871839275222246405745257275088548364400416034343698204186575808495617", 10)
		val.Mod(val, q)
		
		fmt.Printf("        roundConstants[%d] = %s;\n", i, val.String())
		
		hash.Reset()
		hash.Write(rnd)
	}

	fmt.Println(`        return roundConstants;
    }

    function encrypt(uint256 m, uint256 k) internal pure returns (uint256) {
        uint256[110] memory c = getConstants();
        uint256 tmp;
        for (uint256 i = 0; i < 110; i++) {
            // tmp = m + k + c[i]
            tmp = addmod(m, k, Q);
            tmp = addmod(tmp, c[i], Q);
            
            // m = tmp^5
            uint256 tmp2 = mulmod(tmp, tmp, Q);
            uint256 tmp4 = mulmod(tmp2, tmp2, Q);
            m = mulmod(tmp4, tmp, Q);
        }
        m = addmod(m, k, Q);
        return m;
    }

    function hash(uint256[] memory data) public pure returns (uint256) {
        uint256 h = 0;
        for (uint256 i = 0; i < data.length; i++) {
            uint256 r = encrypt(data[i], h);
            h = addmod(addmod(r, h, Q), data[i], Q);
        }
        return h;
    }
}
`)
}
