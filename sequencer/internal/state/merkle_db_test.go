package state

import (
	"math/big"
	"os"
	"testing"

	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestMerkleDB_UpdateAndRoot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "leveldb_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	db, err := NewLevelDBMerkleTree(tmpDir, 28)
	assert.NoError(t, err)
	defer db.Close()

	// Initial root should be the hash of the empty tree
	initialRoot := db.Root()
	assert.NotNil(t, initialRoot)
	assert.True(t, initialRoot.Cmp(big.NewInt(0)) > 0) // Cannot be zero!

	// Create a mock balance state
	pubX := big.NewInt(12345)
	pubY := big.NewInt(67890)
	balance := big.NewInt(1000)
	
	acc := &types.BalanceState{
		AccountID: 1,
		TokenID:   2,
		PubKeyX:   pubX,
		PubKeyY:   pubY,
		Balance:   balance,
		Nonce:     0,
	}

	err = db.SetBalance(acc)
	assert.NoError(t, err)

	// Root should change
	newRoot := db.Root()
	assert.NotEqual(t, initialRoot.String(), newRoot.String())

	// Verify path
	leafIndex := (uint64(1) * 256) + 2
	_, bits := db.GetPath(leafIndex)
	
	// Bits should match the binary representation of leafIndex
	for i := 0; i < 28; i++ {
		expectedBit := (leafIndex >> i) & 1
		assert.Equal(t, expectedBit, bits[i])
	}

	// Verify we can retrieve it
	retrievedAcc, err := db.GetBalance(1, 2)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedAcc)
	assert.Equal(t, balance, retrievedAcc.Balance)
}

func TestMerkleDB_GetNextAccountID(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "leveldb_test2")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	db, err := NewLevelDBMerkleTree(tmpDir, 28)
	assert.NoError(t, err)
	defer db.Close()

	id1, err := db.GetAccountID("0xABCD")
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), id1)

	id2, err := db.GetAccountID("0x1234")
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), id2)

	// Fetching existing account should return the same ID
	id1_again, err := db.GetAccountID("ABCD") // Testing without 0x
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), id1_again)
}
