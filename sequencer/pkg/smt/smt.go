package smt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
)

// SparseMerkleTree is a placeholder implementation of a Sparse Merkle Tree
// This is a simplified version for Milestone 1.4 - will be enhanced later
type SparseMerkleTree struct {
	mu     sync.RWMutex
	leaves map[string][]byte // key -> value mapping
	root   []byte            // current root hash
}

// NewSparseMerkleTree creates a new Sparse Merkle Tree
func NewSparseMerkleTree() *SparseMerkleTree {
	// Initialize with empty root (all zeros)
	emptyRoot := make([]byte, 32)
	return &SparseMerkleTree{
		leaves: make(map[string][]byte),
		root:   emptyRoot,
	}
}

// Update updates a leaf value in the tree and recalculates the root
func (smt *SparseMerkleTree) Update(key string, value []byte) error {
	smt.mu.Lock()
	defer smt.mu.Unlock()

	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	// Store the leaf value
	smt.leaves[key] = value

	// Recalculate root (simplified - will be replaced with proper SMT implementation)
	smt.root = smt.computeRoot()

	return nil
}

// Get retrieves a leaf value from the tree
func (smt *SparseMerkleTree) Get(key string) ([]byte, bool) {
	smt.mu.RLock()
	defer smt.mu.RUnlock()

	value, exists := smt.leaves[key]
	return value, exists
}

// Delete removes a leaf from the tree
func (smt *SparseMerkleTree) Delete(key string) error {
	smt.mu.Lock()
	defer smt.mu.Unlock()

	if _, exists := smt.leaves[key]; !exists {
		return fmt.Errorf("key not found: %s", key)
	}

	delete(smt.leaves, key)

	// Recalculate root
	smt.root = smt.computeRoot()

	return nil
}

// Root returns the current root hash of the tree
func (smt *SparseMerkleTree) Root() []byte {
	smt.mu.RLock()
	defer smt.mu.RUnlock()

	// Return a copy to prevent external modification
	rootCopy := make([]byte, len(smt.root))
	copy(rootCopy, smt.root)
	return rootCopy
}

// RootHex returns the current root hash as a hex string (0x prefixed)
func (smt *SparseMerkleTree) RootHex() string {
	return "0x" + hex.EncodeToString(smt.Root())
}

// computeRoot computes the root hash from all leaves
// This is a placeholder implementation - proper SMT would use Merkle tree structure
func (smt *SparseMerkleTree) computeRoot() []byte {
	if len(smt.leaves) == 0 {
		// Return empty root (all zeros)
		return make([]byte, 32)
	}

	// Simplified: hash all key-value pairs together
	// TODO: Replace with proper Sparse Merkle Tree implementation
	data := make([]byte, 0)
	for key, value := range smt.leaves {
		data = append(data, []byte(key)...)
		data = append(data, value...)
	}

	hash := sha256.Sum256(data)
	return hash[:]
}

// Count returns the number of leaves in the tree
func (smt *SparseMerkleTree) Count() int {
	smt.mu.RLock()
	defer smt.mu.RUnlock()
	return len(smt.leaves)
}

// HasKey checks if a key exists in the tree
func (smt *SparseMerkleTree) HasKey(key string) bool {
	smt.mu.RLock()
	defer smt.mu.RUnlock()
	_, exists := smt.leaves[key]
	return exists
}

// Clear removes all leaves from the tree
func (smt *SparseMerkleTree) Clear() {
	smt.mu.Lock()
	defer smt.mu.Unlock()

	smt.leaves = make(map[string][]byte)
	smt.root = make([]byte, 32) // Reset to empty root
}

