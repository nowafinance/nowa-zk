package api

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/consensys/gnark-crypto/ecc/bn254/twistededwards/eddsa"
	"github.com/nowafinance/nowa-zk/sequencer/internal/batcher"
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

// inactiveLegUpdate mirrors cmd/sequencer/main.go's getEmptyStateUpdate — duplicated
// (not imported) because that lives in package main. Builds the all-zero StateUpdate
// the circuit expects for a deposit's inactive Maker/TakerQuote legs; its Select-based
// gating means the values are unconstrained as long as Index resolves to a real path.
func inactiveLegUpdate(tree *state.LevelDBMerkleTree, index uint64) types.StateUpdate {
	path, bits := tree.GetPath(index)
	var pathStr [28]string
	for i := 0; i < 28; i++ {
		pathStr[i] = path[i].String()
	}
	return types.StateUpdate{Index: index, Balance: "0", Nonce: 0, Path: pathStr, PathBits: bits}
}

// openBalance returns an account's balance leaf, lab-crediting it (1_000_000 test
// units) the first time a pubkey is seen for this token.
//
// The credit is applied as a real, batcher-tracked OpDeposit transition — the exact
// same shape deposit_watcher.go uses for genuine L1 deposits — rather than a direct,
// untracked tree write. A direct write here used to silently advance the Merkle root
// with no corresponding StateTransition, invisible to the batcher and Prover: if a
// new account appeared between two already-sealed batches, the later batch's old_root
// silently stopped matching the earlier batch's new_root, permanently breaking that
// batch with no recovery once batchCount > 0. Hit live: batch #1 settled, batch #2
// never could. See docs/architecture/overview.md's Known Gaps and
// sequencer/cmd/cli/test_client.go's lock-file workaround, which this fix makes
// unnecessary going forward (kept in place regardless — harmless once this is fixed).
func openBalance(tree *state.LevelDBMerkleTree, batch *batcher.Batcher, pubKeyHex string, tokenID uint32) (*types.BalanceState, error) {
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

	leafIndex := (accID * 256) + uint64(tokenID)
	oldRoot := tree.Root().String()
	takerBaseUpdate := inactiveLegUpdate(tree, leafIndex) // pre-credit state: balance 0
	// This leaf has never been written before (acc == nil was just confirmed above) —
	// its true tree value is the SMT's literal 0, not accountLeaf(index,pubX,pubY,0,0).
	// See prover/circuits/state_circuit.go's StateUpdate.IsGenesis doc comment for why
	// conflating the two fails circuit verification (confirmed live on Sepolia).
	takerBaseUpdate.IsGenesis = true

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
	newRoot := tree.Root().String()

	dummyUpdate := inactiveLegUpdate(tree, 99)
	st := types.StateTransition{
		OpType:      batcher.OpDeposit,
		Amount:      "1000000",
		QuoteAmount: "0",

		// Maker is inactive for deposits — same dummy convention deposit_watcher.go
		// uses; fix for the circuit accepting this without a real signature lives in
		// prover/circuits/state_circuit.go.
		MakerPubKeyX: "0",
		MakerPubKeyY: "0",
		MakerSig:     "0x0000000000000000000000000000000000000000000000000000000000000000",
		MakerBase:    dummyUpdate,
		MakerQuote:   dummyUpdate,

		TakerPubKeyX: pubX.String(),
		TakerPubKeyY: pubY.String(),
		TakerSig:     "0x0000000000000000000000000000000000000000000000000000000000000000",
		TakerBase:    takerBaseUpdate,
		TakerQuote:   dummyUpdate,
	}
	batch.AddTransition(st, oldRoot, newRoot)

	return acc, nil
}
