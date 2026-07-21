package smt

import (
	"fmt"
	"testing"
)

func TestNewSparseMerkleTree(t *testing.T) {
	smt := NewSparseMerkleTree()
	if smt == nil {
		t.Fatal("NewSparseMerkleTree returned nil")
	}

	root := smt.Root()
	if len(root) != 32 {
		t.Errorf("Expected root length 32, got %d", len(root))
	}

	// Empty tree should have zero root
	allZeros := true
	for _, b := range root {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if !allZeros {
		t.Error("Empty tree should have zero root")
	}
}

func TestUpdate(t *testing.T) {
	smt := NewSparseMerkleTree()

	key := "test_key"
	value := []byte("test_value")

	err := smt.Update(key, value)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	retrieved, exists := smt.Get(key)
	if !exists {
		t.Fatal("Key should exist after Update")
	}

	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}

	if smt.Count() != 1 {
		t.Errorf("Expected count 1, got %d", smt.Count())
	}
}

func TestGet(t *testing.T) {
	smt := NewSparseMerkleTree()

	key := "test_key"
	value := []byte("test_value")

	// Get non-existent key
	_, exists := smt.Get(key)
	if exists {
		t.Error("Key should not exist")
	}

	// Update and get
	smt.Update(key, value)
	retrieved, exists := smt.Get(key)
	if !exists {
		t.Fatal("Key should exist")
	}

	if string(retrieved) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(retrieved))
	}
}

func TestDelete(t *testing.T) {
	smt := NewSparseMerkleTree()

	key := "test_key"
	value := []byte("test_value")

	// Delete non-existent key
	err := smt.Delete(key)
	if err == nil {
		t.Error("Delete should fail for non-existent key")
	}

	// Update and delete
	smt.Update(key, value)
	err = smt.Delete(key)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, exists := smt.Get(key)
	if exists {
		t.Error("Key should not exist after Delete")
	}

	if smt.Count() != 0 {
		t.Errorf("Expected count 0, got %d", smt.Count())
	}
}

func TestRoot(t *testing.T) {
	smt := NewSparseMerkleTree()

	// Empty root
	root1 := smt.Root()
	if len(root1) != 32 {
		t.Errorf("Expected root length 32, got %d", len(root1))
	}

	// Update and check root changes
	smt.Update("key1", []byte("value1"))
	root2 := smt.Root()

	// Root should change after update
	rootsEqual := true
	for i := range root1 {
		if root1[i] != root2[i] {
			rootsEqual = false
			break
		}
	}
	if rootsEqual {
		t.Error("Root should change after update")
	}

	// RootHex should be prefixed with 0x
	rootHex := smt.RootHex()
	if len(rootHex) != 66 { // 0x + 64 hex chars
		t.Errorf("Expected root hex length 66, got %d", len(rootHex))
	}
	if rootHex[:2] != "0x" {
		t.Error("RootHex should start with 0x")
	}
}

func TestHasKey(t *testing.T) {
	smt := NewSparseMerkleTree()

	key := "test_key"
	if smt.HasKey(key) {
		t.Error("Key should not exist")
	}

	smt.Update(key, []byte("value"))
	if !smt.HasKey(key) {
		t.Error("Key should exist")
	}
}

func TestConcurrentAccess(t *testing.T) {
	smt := NewSparseMerkleTree()

	// Test concurrent updates
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			key := fmt.Sprintf("key_%d", idx)
			smt.Update(key, []byte("value"))
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	if smt.Count() != 10 {
		t.Errorf("Expected count 10, got %d", smt.Count())
	}
}

func TestClear(t *testing.T) {
	smt := NewSparseMerkleTree()

	// Add some leaves
	smt.Update("key1", []byte("value1"))
	smt.Update("key2", []byte("value2"))

	if smt.Count() != 2 {
		t.Errorf("Expected count 2, got %d", smt.Count())
	}

	// Clear
	smt.Clear()

	if smt.Count() != 0 {
		t.Errorf("Expected count 0 after Clear, got %d", smt.Count())
	}

	// Root should be zero
	root := smt.Root()
	allZeros := true
	for _, b := range root {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if !allZeros {
		t.Error("Root should be zero after Clear")
	}
}

