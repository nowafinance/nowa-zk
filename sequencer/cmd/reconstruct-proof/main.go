// reconstruct-proof rebuilds the L2 Merkle tree purely from L1 data — no live
// Sequencer required — and serves the same proof shape GET /proof does, for use
// with claim-escape's --proof-file flag. This is the fully-offline half of the
// escape hatch: claim-escape + a live Sequencer's /proof endpoint covers "the
// Prover died"; this tool covers "the Sequencer itself is gone."
//
// Batch data is fetched from Blobscan (default: Sepolia), not a beacon node — see
// docs/architecture/overview.md and docs/project/release-status.md for why: it
// indexes blobs durably (IPFS-backed, unaffected by the consensus layer's ~18-day
// blob pruning window) and needs no beacon API access at all.
package main

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/nowafinance/nowa-zk/sequencer/internal/state"
)

const lastReplayedBatchKey = "last_replayed_batch"

func main() {
	rollupAddr := flag.String("rollup", "", "NowaRollup contract address (required)")
	rpcURL := flag.String("rpc-url", "", "L1 RPC URL (defaults to $L1_RPC_URL)")
	blobscanAPI := flag.String("blobscan-api", "https://api.sepolia.blobscan.com", "Blobscan API base URL (use https://api.blobscan.com for mainnet)")
	dataDir := flag.String("data-dir", "", "Local checkpoint tree directory (defaults to ~/.nowa-zk/reconstruct-proof/<rollup address>)")
	pubkey := flag.String("pubkey", "", "Your compressed EdDSA pubkey hex (required)")
	tokenID := flag.Uint("token-id", 1, "Token ID to build a proof for")
	outFile := flag.String("out", "", "Write the proof JSON here instead of printing to stdout")
	flag.Parse()

	if *rollupAddr == "" {
		fmt.Println("❌ --rollup is required")
		os.Exit(1)
	}
	if *pubkey == "" {
		fmt.Println("❌ --pubkey is required")
		os.Exit(1)
	}
	if *rpcURL == "" {
		*rpcURL = os.Getenv("L1_RPC_URL")
	}
	if *rpcURL == "" {
		fmt.Println("❌ --rpc-url is required (or set $L1_RPC_URL)")
		os.Exit(1)
	}
	if *dataDir == "" {
		home, _ := os.UserHomeDir()
		*dataDir = filepath.Join(home, ".nowa-zk", "reconstruct-proof", strings.ToLower(*rollupAddr))
	}

	if err := run(*rollupAddr, *rpcURL, *blobscanAPI, *dataDir, *pubkey, uint32(*tokenID), *outFile); err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}
}

func run(rollupAddr, rpcURL, blobscanAPI, dataDir, pubkey string, tokenID uint32, outFile string) error {
	fmt.Printf("--- Opening local checkpoint tree at %s ---\n", dataDir)
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	tree, err := state.NewLevelDBMerkleTree(dataDir, 28)
	if err != nil {
		return fmt.Errorf("open local tree: %w", err)
	}
	defer tree.Close()

	reader, err := newRollupReader(rpcURL, rollupAddr)
	if err != nil {
		return fmt.Errorf("connect to L1: %w", err)
	}

	if err := replayAllBatches(tree, reader, blobscanAPI); err != nil {
		return err
	}

	if err := verifyAgainstOnChainRoot(tree, reader); err != nil {
		return err
	}

	proof, err := buildProof(tree, pubkey, tokenID)
	if err != nil {
		return fmt.Errorf("build proof: %w", err)
	}

	out, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal proof: %w", err)
	}
	if outFile != "" {
		if err := os.WriteFile(outFile, out, 0644); err != nil {
			return fmt.Errorf("write %s: %w", outFile, err)
		}
		fmt.Printf("✅ Proof written to %s\n", outFile)
	} else {
		fmt.Println("\n--- Proof ---")
		fmt.Println(string(out))
	}
	return nil
}

// readCheckpoint returns the next batch ID to replay (0 if none has been replayed
// yet). Factored out of replayAllBatches so the corrupt-data case — a truncated or
// otherwise malformed persisted value — is independently testable without needing a
// live rollupReader/L1 connection.
func readCheckpoint(tree *state.LevelDBMerkleTree) (uint64, error) {
	data, ok, err := tree.GetMeta(lastReplayedBatchKey)
	if err != nil {
		return 0, fmt.Errorf("read checkpoint: %w", err)
	}
	if !ok {
		return 0, nil
	}
	if len(data) != 8 {
		return 0, fmt.Errorf("corrupt checkpoint value: expected 8 bytes, got %d", len(data))
	}
	return binary.BigEndian.Uint64(data) + 1, nil
}

func replayAllBatches(tree *state.LevelDBMerkleTree, reader *rollupReader, blobscanAPI string) error {
	checkpoint, err := readCheckpoint(tree)
	if err != nil {
		return err
	}

	batchCount, err := reader.batchCount()
	if err != nil {
		return fmt.Errorf("read batchCount from L1: %w", err)
	}
	fmt.Printf("--- Replaying batches %d..%d (on-chain batchCount=%d) ---\n", checkpoint, batchCount, batchCount)

	for i := checkpoint; i < batchCount; i++ {
		if err := replayOneBatch(tree, reader, blobscanAPI, i); err != nil {
			return fmt.Errorf("batch #%d: %w", i, err)
		}
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, i)
		if err := tree.SetMeta(lastReplayedBatchKey, buf); err != nil {
			return fmt.Errorf("save checkpoint after batch #%d: %w", i, err)
		}
		fmt.Printf("  ✅ batch #%d replayed and root-verified\n", i)
	}
	return nil
}

func replayOneBatch(tree *state.LevelDBMerkleTree, reader *rollupReader, blobscanAPI string, batchID uint64) error {
	blobHash, err := reader.batchBlobHash(batchID)
	if err != nil {
		return fmt.Errorf("read batchBlobHash: %w", err)
	}
	dataHash, err := reader.batchDataHash(batchID)
	if err != nil {
		return fmt.Errorf("read batchDataHash: %w", err)
	}

	versionedHash := "0x" + hex.EncodeToString(blobHash[:])
	blob, err := fetchBlob(blobscanAPI, versionedHash)
	if err != nil {
		return fmt.Errorf("fetch blob from Blobscan: %w", err)
	}

	payloadBytes, err := unpackBlob(blob)
	if err != nil {
		return fmt.Errorf("unpack blob: %w", err)
	}

	// Integrity check #1: the payload must hash to what the contract recorded —
	// don't trust Blobscan's content without this.
	gotHash := crypto.Keccak256Hash(payloadBytes)
	if gotHash != dataHash {
		return fmt.Errorf("dataHash mismatch: blob content doesn't match what NowaRollup recorded (got %s, want 0x%x) — Blobscan data may be corrupt or wrong", gotHash.Hex(), dataHash)
	}

	payload, err := decodeDAPayload(payloadBytes)
	if err != nil {
		return err
	}

	for i, st := range payload.Transitions {
		if err := replayTransition(tree, st); err != nil {
			return fmt.Errorf("transition #%d: %w", i, err)
		}
	}

	// Integrity check #2: the replayed root must match what this batch's proof
	// actually committed to — catches a replay-arithmetic bug immediately rather
	// than silently drifting into later batches.
	wantRoot, ok := new(big.Int).SetString(payload.NewRoot, 10)
	if !ok {
		return fmt.Errorf("invalid new_root %q in payload", payload.NewRoot)
	}
	if tree.Root().Cmp(wantRoot) != 0 {
		return fmt.Errorf("replayed root doesn't match batch's new_root:\n  got:  %s\n  want: %s", tree.Root().String(), wantRoot.String())
	}
	return nil
}

func verifyAgainstOnChainRoot(tree *state.LevelDBMerkleTree, reader *rollupReader) error {
	onChainRoot, err := reader.stateRoot()
	if err != nil {
		return fmt.Errorf("read current stateRoot from L1: %w", err)
	}
	want := new(big.Int).SetBytes(onChainRoot[:])
	if tree.Root().Cmp(want) != 0 {
		return fmt.Errorf("fully-replayed local root does NOT match the contract's current stateRoot:\n  local: %s\n  chain: %s\n(this means either replay is buggy, or the contract has settled a batch this tool couldn't fetch — do not trust the proof below)", tree.Root().String(), want.String())
	}
	fmt.Println("--- Local root matches on-chain stateRoot — reconstruction verified ---")
	return nil
}

// proofOutput mirrors GET /proof's JSON shape exactly (sequencer/internal/api/server.go),
// so it's a drop-in for claim-escape --proof-file.
type proofOutput struct {
	PubKey    string   `json:"pubkey"`
	AccountID uint64   `json:"account_id"`
	TokenID   uint32   `json:"token_id"`
	Index     uint64   `json:"index"`
	Balance   string   `json:"balance"`
	Nonce     uint64   `json:"nonce"`
	PubKeyX   string   `json:"pub_key_x"`
	PubKeyY   string   `json:"pub_key_y"`
	Siblings  []string `json:"siblings"`
	PathBits  []int    `json:"path_bits"`
}

func buildProof(tree *state.LevelDBMerkleTree, pubkey string, tokenID uint32) (*proofOutput, error) {
	accID, err := tree.GetAccountID(pubkey)
	if err != nil {
		return nil, err
	}
	acc, err := tree.GetBalance(accID, tokenID)
	if err != nil {
		return nil, err
	}

	balance := "0"
	nonce := uint64(0)
	pubX, pubY := "0", "0"
	if acc != nil {
		balance = acc.Balance.String()
		nonce = acc.Nonce
		pubX = acc.PubKeyX.String()
		pubY = acc.PubKeyY.String()
	}

	index := (accID * 256) + uint64(tokenID)
	path, bits := tree.GetPath(index)
	siblings := make([]string, 28)
	pathBits := make([]int, 28)
	for i := 0; i < 28; i++ {
		siblings[i] = path[i].String()
		pathBits[i] = int(bits[i])
	}

	return &proofOutput{
		PubKey: pubkey, AccountID: accID, TokenID: tokenID, Index: index,
		Balance: balance, Nonce: nonce, PubKeyX: pubX, PubKeyY: pubY,
		Siblings: siblings, PathBits: pathBits,
	}, nil
}
