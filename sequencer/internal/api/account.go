package api

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
)

func decodePubKeyXY(pubKeyHex string) (*big.Int, *big.Int, error) {
	clean := strings.TrimPrefix(pubKeyHex, "0x")
	raw, err := hex.DecodeString(clean)
	if err != nil {
		return nil, nil, err
	}
	var pk eddsa.PublicKey
	if _, err := pk.SetBytes(raw); err != nil {
		return nil, nil, err
	}
	x := new(big.Int)
	y := new(big.Int)
	pk.A.X.BigInt(x)
	pk.A.Y.BigInt(y)
	return x, y, nil
}

// openBalance creates a lab balance leaf if missing (1_000_000 test units).
func openBalance(tree *state.LevelDBMerkleTree, pubKeyHex string, tokenID uint32) (*types.BalanceState, error) {
	accID, err := tree.GetAccountID(pubKeyHex)
	if err != nil {
		return nil, err
	}
	acc, err := tree.GetBalance(accID, tokenID)
	if err != nil {
		return nil, err
	}
	if acc != nil {
		return acc, nil
	}
	pubX, pubY, err := decodePubKeyXY(pubKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid pubkey: %w", err)
	}
	acc = &types.BalanceState{
		AccountID: accID,
		TokenID:   tokenID,
		PubKeyX:   pubX,
		PubKeyY:   pubY,
		Balance:   big.NewInt(1000000),
		Nonce:     0,
	}
	if err := tree.SetBalance(acc); err != nil {
		return nil, err
	}
	return acc, nil
}
