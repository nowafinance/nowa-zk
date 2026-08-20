// claim-escape is the escape-hatch client: it fetches a Merkle proof for an L2
// account from a running Sequencer's GET /proof and submits it to
// NowaRollup.emergencyWithdraw() on L1. This is the actual "bridge" a user runs when
// the Prover has stalled past escapeTimeout — see docs/architecture/overview.md's
// Known Gaps and docs/project/release-status.md for the full context on why this
// tool exists and what it does/doesn't cover (deposit-bound escape hatch scope).
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Only the one function this tool calls — deliberately not a full ABI/binding
// regeneration (the checked-in sequencer/internal/bindings/nowa_rollup.go is
// already known-stale for other functions; see release-status.md). Mirrors the
// same "hand-write the ABI JSON for what you need" pattern prover/cmd/prover/start.go
// already uses for submitBatch.
const emergencyWithdrawABI = `[{
	"inputs": [{
		"components": [
			{"internalType":"uint32","name":"tokenId","type":"uint32"},
			{"internalType":"uint256","name":"balance","type":"uint256"},
			{"internalType":"uint256","name":"nonce","type":"uint256"},
			{"internalType":"uint256","name":"pubX","type":"uint256"},
			{"internalType":"uint256","name":"pubY","type":"uint256"},
			{"internalType":"uint256","name":"index","type":"uint256"},
			{"internalType":"bytes32[28]","name":"siblings","type":"bytes32[28]"},
			{"internalType":"bool[28]","name":"pathBits","type":"bool[28]"}
		],
		"internalType": "struct NowaRollup.EscapeProof",
		"name": "p",
		"type": "tuple"
	}],
	"name": "emergencyWithdraw",
	"outputs": [],
	"stateMutability": "nonpayable",
	"type": "function"
}]`

// proofResponse mirrors GET /proof's JSON shape (sequencer/internal/api/server.go's handleProof).
type proofResponse struct {
	Index    uint64   `json:"index"`
	Balance  string   `json:"balance"`
	Nonce    uint64   `json:"nonce"`
	PubKeyX  string   `json:"pub_key_x"`
	PubKeyY  string   `json:"pub_key_y"`
	Siblings []string `json:"siblings"`
	PathBits []int    `json:"path_bits"`
}

// escapeProof's field names/order must match NowaRollup.sol's EscapeProof struct —
// go-ethereum's abi.Pack matches Go struct fields to ABI tuple components by name.
type escapeProof struct {
	TokenId  uint32
	Balance  *big.Int
	Nonce    *big.Int
	PubX     *big.Int
	PubY     *big.Int
	Index    *big.Int
	Siblings [28][32]byte
	PathBits [28]bool
}

func main() {
	sequencerURL := flag.String("sequencer-url", "http://localhost:8080", "Sequencer REST API base URL (ignored if --proof-file is set)")
	proofFile := flag.String("proof-file", "", "Read the proof from this JSON file instead of a live Sequencer — use output from sequencer/cmd/reconstruct-proof when the Sequencer itself is offline")
	rpcURL := flag.String("rpc-url", "", "L1 RPC URL (defaults to $L1_RPC_URL)")
	contractAddr := flag.String("contract", "", "NowaRollup address (defaults to ~/.nowa-zk/deployments.json)")
	privateKeyHex := flag.String("private-key", "", "Depositor's private key (defaults to $PRIVATE_KEY) — must be the address that called deposit() for this pubkey")
	pubkey := flag.String("pubkey", "", "Your compressed EdDSA pubkey hex, e.g. 0xabc... (required)")
	tokenID := flag.Uint("token-id", 1, "Token ID to withdraw")
	dryRun := flag.Bool("dry-run", false, "Fetch and print the proof + would-be call without sending a transaction")
	printCalldata := flag.Bool("print-calldata", false, "Fetch the proof, print the raw ABI-encoded calldata as hex, and exit without sending")
	flag.Parse()

	if *pubkey == "" {
		fmt.Println("❌ --pubkey is required (your compressed EdDSA pubkey, the same one you traded with)")
		os.Exit(1)
	}
	if *rpcURL == "" {
		*rpcURL = os.Getenv("L1_RPC_URL")
	}
	if *privateKeyHex == "" {
		*privateKeyHex = os.Getenv("PRIVATE_KEY")
	}
	if *contractAddr == "" {
		*contractAddr = loadContractFromDeployments()
	}

	var proof *proofResponse
	var err error
	if *proofFile != "" {
		fmt.Printf("--- Reading proof from %s ---\n", *proofFile)
		proof, err = readProofFile(*proofFile)
	} else {
		fmt.Printf("--- Fetching proof from %s ---\n", *sequencerURL)
		proof, err = fetchProof(*sequencerURL, *pubkey, uint32(*tokenID))
	}
	if err != nil {
		fmt.Printf("❌ Failed to get proof: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  index=%d balance=%s nonce=%d\n", proof.Index, proof.Balance, proof.Nonce)

	ep, err := toEscapeProof(proof, uint32(*tokenID))
	if err != nil {
		fmt.Printf("❌ Failed to build proof struct: %v\n", err)
		os.Exit(1)
	}

	if *dryRun {
		fmt.Println("\n--dry-run set — not sending. Would call emergencyWithdraw with:")
		printProof(ep)
		return
	}

	if *printCalldata {
		parsedABI, err := abi.JSON(strings.NewReader(emergencyWithdrawABI))
		if err != nil {
			fmt.Println("abi parse err:", err)
			os.Exit(1)
		}
		packed, err := parsedABI.Pack("emergencyWithdraw", ep)
		if err != nil {
			fmt.Println("pack err:", err)
			os.Exit(1)
		}
		fmt.Println("CALLDATA=0x" + hex.EncodeToString(packed))
		return
	}

	if *rpcURL == "" {
		fmt.Println("❌ No RPC URL — pass --rpc-url or set L1_RPC_URL")
		os.Exit(1)
	}
	if *privateKeyHex == "" {
		fmt.Println("❌ No private key — pass --private-key or set PRIVATE_KEY")
		os.Exit(1)
	}
	if *contractAddr == "" {
		fmt.Println("❌ No contract address — pass --contract or deploy so ~/.nowa-zk/deployments.json exists")
		os.Exit(1)
	}

	fmt.Printf("\n--- Submitting emergencyWithdraw to %s ---\n", *contractAddr)
	txHash, err := submit(*rpcURL, *privateKeyHex, *contractAddr, ep)
	if err != nil {
		fmt.Printf("❌ Transaction failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Submitted: %s\n", txHash)
}

func loadContractFromDeployments() string {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".nowa-zk", "deployments.json"))
	if err != nil {
		return ""
	}
	var deployments map[string]string
	if err := json.Unmarshal(data, &deployments); err != nil {
		return ""
	}
	return deployments["NowaRollup"]
}

func fetchProof(sequencerURL, pubkey string, tokenID uint32) (*proofResponse, error) {
	url := fmt.Sprintf("%s/proof?pubkey=%s&token_id=%d", sequencerURL, pubkey, tokenID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("connect to Sequencer: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Sequencer returned %s: %s", resp.Status, string(body))
	}
	var pr proofResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("decode response: %w (body: %s)", err, body)
	}
	if len(pr.Siblings) != 28 || len(pr.PathBits) != 28 {
		return nil, fmt.Errorf("expected 28 siblings/path_bits, got %d/%d — is the Sequencer running the current build?", len(pr.Siblings), len(pr.PathBits))
	}
	return &pr, nil
}

// readProofFile loads a proof from disk — the same JSON shape fetchProof returns,
// as produced by sequencer/cmd/reconstruct-proof when there's no live Sequencer to ask.
func readProofFile(path string) (*proofResponse, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var pr proofResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	if len(pr.Siblings) != 28 || len(pr.PathBits) != 28 {
		return nil, fmt.Errorf("%s: expected 28 siblings/path_bits, got %d/%d", path, len(pr.Siblings), len(pr.PathBits))
	}
	return &pr, nil
}

func toEscapeProof(pr *proofResponse, tokenID uint32) (escapeProof, error) {
	var ep escapeProof
	ep.TokenId = tokenID

	balance, ok := new(big.Int).SetString(pr.Balance, 10)
	if !ok {
		return ep, fmt.Errorf("invalid balance %q", pr.Balance)
	}
	ep.Balance = balance
	ep.Nonce = new(big.Int).SetUint64(pr.Nonce)

	pubX, ok := new(big.Int).SetString(pr.PubKeyX, 10)
	if !ok {
		return ep, fmt.Errorf("invalid pub_key_x %q", pr.PubKeyX)
	}
	ep.PubX = pubX
	pubY, ok := new(big.Int).SetString(pr.PubKeyY, 10)
	if !ok {
		return ep, fmt.Errorf("invalid pub_key_y %q", pr.PubKeyY)
	}
	ep.PubY = pubY
	ep.Index = new(big.Int).SetUint64(pr.Index)

	for i := 0; i < 28; i++ {
		s, ok := new(big.Int).SetString(pr.Siblings[i], 10)
		if !ok {
			return ep, fmt.Errorf("invalid sibling[%d] %q", i, pr.Siblings[i])
		}
		s.FillBytes(ep.Siblings[i][:])
		ep.PathBits[i] = pr.PathBits[i] == 1
	}
	return ep, nil
}

func printProof(ep escapeProof) {
	fmt.Printf("  tokenId=%d balance=%s nonce=%s pubX=%s pubY=%s index=%s\n",
		ep.TokenId, ep.Balance, ep.Nonce, ep.PubX, ep.PubY, ep.Index)
	fmt.Printf("  siblings[0]=0x%s ... siblings[27]=0x%s\n",
		hex.EncodeToString(ep.Siblings[0][:]), hex.EncodeToString(ep.Siblings[27][:]))
}

func submit(rpcURL, privateKeyHex, contractAddr string, ep escapeProof) (string, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return "", fmt.Errorf("dial L1 RPC: %w", err)
	}

	priv, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return "", fmt.Errorf("get chain ID: %w", err)
	}
	auth, err := bind.NewKeyedTransactorWithChainID(priv, chainID)
	if err != nil {
		return "", fmt.Errorf("build transactor: %w", err)
	}

	parsedABI, err := abi.JSON(strings.NewReader(emergencyWithdrawABI))
	if err != nil {
		return "", fmt.Errorf("parse ABI: %w", err)
	}
	bound := bind.NewBoundContract(common.HexToAddress(contractAddr), parsedABI, client, client, client)

	tx, err := bound.Transact(auth, "emergencyWithdraw", ep)
	if err != nil {
		return "", fmt.Errorf("send transaction: %w", err)
	}

	fmt.Println("  waiting for confirmation...")
	receipt, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		return tx.Hash().Hex(), fmt.Errorf("wait for receipt: %w", err)
	}
	if receipt.Status == 0 {
		return tx.Hash().Hex(), fmt.Errorf("transaction reverted (block %d)", receipt.BlockNumber)
	}
	return tx.Hash().Hex(), nil
}
