package state

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/nowafinance/nowa-zk/sequencer/internal/types"
	"github.com/syndtr/goleveldb/leveldb"
)

// LevelDBMerkleTree represents a sparse Merkle tree of depth D, backed by LevelDB.
type LevelDBMerkleTree struct {
	db         *leveldb.DB
	depth      int
	zeroHashes []fr.Element
}

// NewLevelDBMerkleTree initializes or opens a LevelDB instance for the state.
func NewLevelDBMerkleTree(dbPath string, depth int) (*LevelDBMerkleTree, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, err
	}

	smt := &LevelDBMerkleTree{
		db:    db,
		depth: depth,
	}
	smt.computeZeroHashes()
	return smt, nil
}

// Close closes the underlying LevelDB.
func (smt *LevelDBMerkleTree) Close() error {
	return smt.db.Close()
}

func (smt *LevelDBMerkleTree) computeZeroHashes() {
	smt.zeroHashes = make([]fr.Element, smt.depth+1)
	smt.zeroHashes[0].SetZero()

	for i := 1; i <= smt.depth; i++ {
		h := mimc.NewMiMC()
		b := smt.zeroHashes[i-1].Bytes()
		h.Write(b[:])
		h.Write(b[:])
		hashBytes := h.Sum(nil)
		smt.zeroHashes[i].SetBytes(hashBytes)
	}
}

func nodeKey(level int, index uint64) []byte {
	k := make([]byte, 12)
	binary.BigEndian.PutUint32(k[0:4], uint32(level))
	binary.BigEndian.PutUint64(k[4:12], index)
	return k
}

// Update updates the leaf at `index` to `leafHash`.
func (smt *LevelDBMerkleTree) Update(index uint64, leafHash *fr.Element) error {
	batch := new(leveldb.Batch)
	
	cur := *leafHash
	curBytes := cur.Bytes()
	batch.Put(nodeKey(0, index), curBytes[:])

	for i := 0; i < smt.depth; i++ {
		siblingIndex := index ^ 1
		sibling := smt.getNode(i, siblingIndex)

		h := mimc.NewMiMC()
		left, right := cur, sibling
		if index%2 == 1 {
			left, right = sibling, cur
		}

		leftBytes := left.Bytes()
		rightBytes := right.Bytes()
		h.Write(leftBytes[:])
		h.Write(rightBytes[:])

		cur.SetBytes(h.Sum(nil))
		index /= 2
		curBytesLoop := cur.Bytes()
		batch.Put(nodeKey(i+1, index), curBytesLoop[:])
	}
	
	return smt.db.Write(batch, nil)
}

// getNode retrieves a node from DB or returns the zero hash.
func (smt *LevelDBMerkleTree) getNode(level int, index uint64) fr.Element {
	data, err := smt.db.Get(nodeKey(level, index), nil)
	if err == leveldb.ErrNotFound {
		return smt.zeroHashes[level]
	}
	var res fr.Element
	res.SetBytes(data)
	return res
}

// GetPath retrieves the Merkle proof for a given leaf index.
func (smt *LevelDBMerkleTree) GetPath(index uint64) ([28]*big.Int, [28]uint64) {
	var path [28]*big.Int
	var bits [28]uint64

	for i := 0; i < smt.depth; i++ {
		siblingIndex := index ^ 1
		sibling := smt.getNode(i, siblingIndex)
		
		siblingBigInt := new(big.Int)
		sibling.BigInt(siblingBigInt)
		path[i] = siblingBigInt
		
		bits[i] = index % 2
		index /= 2
	}
	return path, bits
}

// Root returns the current root hash.
func (smt *LevelDBMerkleTree) Root() *big.Int {
	rootFr := smt.getNode(smt.depth, 0)
	res := new(big.Int)
	rootFr.BigInt(res)
	return res
}

// HashAccountLeaf hashes the account struct using MiMC.
func HashAccountLeaf(index, pubX, pubY, balance, nonce *big.Int) *fr.Element {
	h := mimc.NewMiMC()
	
	var idxFr, pubXFr, pubYFr, balFr, nonceFr fr.Element
	idxFr.SetBigInt(index)
	pubXFr.SetBigInt(pubX)
	pubYFr.SetBigInt(pubY)
	balFr.SetBigInt(balance)
	nonceFr.SetBigInt(nonce)

	idxBytes := idxFr.Bytes()
	pubXBytes := pubXFr.Bytes()
	pubYBytes := pubYFr.Bytes()
	balBytes := balFr.Bytes()
	nonceBytes := nonceFr.Bytes()

	h.Write(idxBytes[:])
	h.Write(pubXBytes[:])
	h.Write(pubYBytes[:])
	h.Write(balBytes[:])
	h.Write(nonceBytes[:])

	var res fr.Element
	res.SetBytes(h.Sum(nil))
	return &res
}

// GetBalance retrieves the balance state by AccountID and TokenID.
func (smt *LevelDBMerkleTree) GetBalance(accountID uint64, tokenID uint32) (*types.BalanceState, error) {
	key := fmt.Sprintf("balance:%d:%d", accountID, tokenID)
	data, err := smt.db.Get([]byte(key), nil)
	if err == leveldb.ErrNotFound {
		return nil, nil // balance not found
	} else if err != nil {
		return nil, err
	}

	var acc types.BalanceState
	if err := json.Unmarshal(data, &acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

// SetBalance saves the balance state and automatically updates its leaf in the Merkle Tree!
func (smt *LevelDBMerkleTree) SetBalance(acc *types.BalanceState) error {
	key := fmt.Sprintf("balance:%d:%d", acc.AccountID, acc.TokenID)
	
	// 1. Save to DB
	data, err := json.Marshal(acc)
	if err != nil {
		return err
	}
	if err := smt.db.Put([]byte(key), data, nil); err != nil {
		return err
	}

	leafIndex := (acc.AccountID * 256) + uint64(acc.TokenID)

	// 2. Hash the new balance state
	leafHash := HashAccountLeaf(
		new(big.Int).SetUint64(leafIndex),
		acc.PubKeyX,
		acc.PubKeyY,
		acc.Balance,
		new(big.Int).SetUint64(acc.Nonce),
	)

	// 3. Update the Merkle Tree at the leaf index
	return smt.Update(leafIndex, leafHash)
}

// SetMeta stores an arbitrary metadata value (e.g. a checkpoint for an external
// replay/reconstruction tool), namespaced under "meta:" so it can never collide
// with balance/account/node keys.
func (smt *LevelDBMerkleTree) SetMeta(key string, value []byte) error {
	return smt.db.Put([]byte("meta:"+key), value, nil)
}

// GetMeta retrieves a value previously stored with SetMeta. ok is false if the key
// was never set (not an error — a fresh tree has no metadata yet).
func (smt *LevelDBMerkleTree) GetMeta(key string) (value []byte, ok bool, err error) {
	data, err := smt.db.Get([]byte("meta:"+key), nil)
	if err == leveldb.ErrNotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

// GetNextAccountID gets and increments the global account ID counter.
func (smt *LevelDBMerkleTree) GetNextAccountID() (uint64, error) {
	data, err := smt.db.Get([]byte("next_account_id"), nil)
	var idx uint64 = 0
	if err == nil {
		idx = binary.BigEndian.Uint64(data)
	} else if err != leveldb.ErrNotFound {
		return 0, err
	}

	// increment and save
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, idx+1)
	if err := smt.db.Put([]byte("next_account_id"), buf, nil); err != nil {
		return 0, err
	}
	
	return idx, nil
}

// GetAccountID retrieves an AccountID by PubKey.
func (smt *LevelDBMerkleTree) GetAccountID(pubKeyHex string) (uint64, error) {
	cleanPubKeyHex := strings.TrimPrefix(pubKeyHex, "0x")
	data, err := smt.db.Get([]byte("pubkey:"+cleanPubKeyHex), nil)
	if err == leveldb.ErrNotFound {
		// Create new account ID
		newID, err := smt.GetNextAccountID()
		if err != nil {
			return 0, err
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, newID)
		smt.db.Put([]byte("pubkey:"+cleanPubKeyHex), buf, nil)
		return newID, nil
	} else if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(data), nil
}
