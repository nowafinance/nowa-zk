package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark/backend/groth16"
	bn254 "github.com/consensys/gnark/backend/groth16/bn254"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"github.com/tannetwork/zk-sequencer/prover/circuits"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start prover service to fetch batches and submit proofs",
	Long:  `Fetches transaction batches from sequencer API, generates ZK proofs, and submits to smart contract.`,
	Run:   start,
}

var (
	keysDir       string
	sequencerURL  string
	rpcURL        string
	contractAddr  string
	privateKeyHex string
	pollInterval  int
	configPath    string
)

func init() {
	// Keys directory (default: ./keys)
	startCmd.Flags().StringVarP(&keysDir, "keys-dir", "k", "./keys", "")

	// Sequencer API URL (default: http://localhost:8080)
	startCmd.Flags().StringVarP(&sequencerURL, "sequencer-url", "s", "http://localhost:8080", "")

	// Ethereum RPC URL (default: http://localhost:8545)
	startCmd.Flags().StringVarP(&rpcURL, "rpc-url", "r", "http://localhost:8545", "")

	// Contract address (required)
	startCmd.Flags().StringVarP(&contractAddr, "contract", "c", "", "")

	// Private key for submitting txs (required)
	startCmd.Flags().StringVarP(&privateKeyHex, "private-key", "p", "", "")

	// Poll interval in seconds (default: 10)
	startCmd.Flags().IntVarP(&pollInterval, "poll-interval", "i", 10, "")
	startCmd.Flags().StringVar(&configPath, "config", "", "Path to YAML config file")
}

// Batch represents a batch from sequencer API
type Batch struct {
	Number        uint64        `json:"number"`
	Hash          string        `json:"hash"`
	PrevStateRoot string        `json:"oldStateRoot"`
	NewStateRoot  string        `json:"newStateRoot"`
	Timestamp     int64         `json:"timestamp"`
	Transactions  []Transaction `json:"transactions"`
}

// Transaction from sequencer
type Transaction struct {
	Hash  string      `json:"hash"`
	From  string      `json:"from"`
	To    string      `json:"to"`
	Value json.Number `json:"value"`
	Nonce json.Number `json:"nonce"`
	Data  string      `json:"input"`
}

func start(cmd *cobra.Command, args []string) {
	// 1. Auto-load configuration if missing
	if contractAddr == "" {
		// Try to load from .tan-zk/deployments.json
		if data, err := os.ReadFile(".tan-zk/deployments.json"); err == nil {
			// Simple string parsing to avoid importing complex JSON struct
			// Looking for "BatchRegistry": "0x..."
			content := string(data)
			if idx := strings.Index(content, "\"BatchRegistry\""); idx != -1 {
				rest := content[idx:]
				if start := strings.Index(rest, ":"); start != -1 {
					if quote1 := strings.Index(rest[start:], "\""); quote1 != -1 {
						if quote2 := strings.Index(rest[start+quote1+1:], "\""); quote2 != -1 {
							contractAddr = rest[start+quote1+1 : start+quote1+1+quote2]
							log.Printf("ℹ️  Auto-loaded Contract: %s", contractAddr)
						}
					}
				}
			}
		}
	}

	if privateKeyHex == "" {
		// Try to load from .tan-zk/secrets.env
		if data, err := os.ReadFile(".tan-zk/secrets.env"); err == nil {
			content := string(data)
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRIVATE_KEY=") {
					privateKeyHex = strings.TrimSpace(strings.TrimPrefix(line, "PRIVATE_KEY="))
					log.Println("ℹ️  Auto-loaded Private Key from .tan-zk/secrets.env")
					break
				}
			}
		}
	}

	// Validate required arguments
	if contractAddr == "" {
		log.Fatal("❌ Contract address required. Use --contract flag or ensure .tan-zk/deployments.json exists.")
	}
	if privateKeyHex == "" {
		log.Fatal("❌ Private key required. Use --private-key flag or ensure .tan-zk/secrets.env exists.")
	}
	log.Println("========================================")
	log.Println("  ZK Rollup Prover Service")
	log.Println("========================================")
	log.Printf("Sequencer API: %s\n", sequencerURL)
	log.Printf("RPC URL: %s\n", rpcURL)
	log.Printf("Contract: %s\n", contractAddr)
	log.Printf("Poll interval: %d seconds\n", pollInterval)
	log.Println()

	// Load circuit and keys
	log.Println("📦 Loading circuit and keys...")
	ccs, pk, err := loadCircuitAndKeys()
	if err != nil {
		log.Fatalf("❌ Failed to load circuit/keys: %v", err)
	}
	log.Println("✅ Circuit and keys loaded")
	log.Println()

	// Connect to Ethereum
	log.Println("🔗 Connecting to Ethereum...")
	client, auth, err := connectEthereum(rpcURL, privateKeyHex)
	if err != nil {
		log.Fatalf("❌ Failed to connect to Ethereum: %v", err)
	}
	log.Println("✅ Connected to Ethereum")
	log.Println()

	// Main prover loop
	log.Println("🚀 Starting prover loop...")
	log.Println("   Polling for new batches...")
	log.Println()

	lastProcessedBatch := uint64(0)

	for {
		// 1. Get the latest batch number from the sequencer
		latestBatch, err := fetchLatestBatch(sequencerURL)
		if err != nil {
			log.Printf("⚠️  Failed to fetch latest batch info: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		log.Printf("✅ Fetched latest batch info: %v\n", latestBatch.Number)

		// 2. Determine the next batch we want to process
		nextBatchNum := lastProcessedBatch + 1

		// 3. Check if we are strictly behind the latest batch
		// User wants: "if batches have reached 41, make baches till 40 only"
		// So we can process 'nextBatchNum' ONLY IF nextBatchNum < latestBatch.Number
		if nextBatchNum >= latestBatch.Number {
			// We are caught up to (latest - 1), or there are no new batches to process safely
			log.Printf("Waiting for new batches... (Latest: %d, Processed: %d)\n", latestBatch.Number, lastProcessedBatch)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		// 4. Fetch the specific batch we want to process
		batch, err := fetchBatch(sequencerURL, nextBatchNum)
		if err != nil {
			log.Printf("⚠️  Failed to fetch batch #%d: %v\n", nextBatchNum, err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		log.Printf("📦 Processing batch #%d (%d transactions)\n", batch.Number, len(batch.Transactions))

		// Generate proof
		log.Println("   🔐 Generating proof...")
		proof, publicWitness, err := generateProof(batch, ccs, pk)
		if err != nil {
			log.Printf("   ❌ Failed to generate proof: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		log.Println("   ✅ Proof generated")

		// Submit to contract
		log.Println("   📡 Submitting proof to contract...")
		txHash, err := submitProof(client, auth, contractAddr, batch, proof, publicWitness)
		if err != nil {
			log.Printf("   ❌ Failed to submit proof: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		log.Printf("   ✅ Proof submitted: %s\n", txHash)
		log.Println()

		lastProcessedBatch = batch.Number
	}
}

func loadCircuitAndKeys() (constraint.ConstraintSystem, groth16.ProvingKey, error) {
	// Load compiled circuit
	ccsFile, err := os.Open(keysDir + "/rollup.r1cs")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open circuit file: %w", err)
	}
	defer ccsFile.Close()

	ccs := groth16.NewCS(ecc.BN254)
	if _, err := ccs.ReadFrom(ccsFile); err != nil {
		return nil, nil, fmt.Errorf("failed to read circuit: %w", err)
	}

	// Load proving key
	pkFile, err := os.Open(keysDir + "/rollup.pk")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open proving key: %w", err)
	}
	defer pkFile.Close()

	pk := groth16.NewProvingKey(ecc.BN254)
	if _, err := pk.ReadFrom(pkFile); err != nil {
		return nil, nil, fmt.Errorf("failed to read proving key: %w", err)
	}

	return ccs, pk, nil
}

func connectEthereum(rpcURL, privateKeyHex string) (*ethclient.Client, *bind.TransactOpts, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to ethereum: %w", err)
	}

	// Strip 0x prefix if present
	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid private key: %w", err)
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	return client, auth, nil
}

func fetchLatestBatch(sequencerURL string) (*Batch, error) {
	resp, err := http.Get(sequencerURL + "/prover/batch/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var batch Batch
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return nil, err
	}

	return &batch, nil
}

func fetchBatch(sequencerURL string, number uint64) (*Batch, error) {
	url := fmt.Sprintf("%s/prover/batch/%d", sequencerURL, number)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var batch Batch
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return nil, err
	}

	return &batch, nil
}

func generateProof(batch *Batch, ccs constraint.ConstraintSystem, pk groth16.ProvingKey) (groth16.Proof, witness.Witness, error) {
	// Convert batch to circuit witness
	var circuit circuits.Circuit

	// Fill transactions (pad with zeros if needed)
	for i := 0; i < circuits.BatchSize; i++ {
		if i < len(batch.Transactions) {
			tx := batch.Transactions[i]
			circuit.Transactions[i] = circuits.Transaction{
				From:      hashAddress(tx.From),
				To:        hashAddress(tx.To),
				Amount:    parseBigInt(string(tx.Value)),
				Nonce:     parseBigInt(string(tx.Nonce)),
				InputHash: hashData(tx.Data),
			}
		} else {
			// Pad with zero transactions
			circuit.Transactions[i] = circuits.Transaction{
				From:      big.NewInt(0),
				To:        big.NewInt(0),
				Amount:    big.NewInt(0),
				Nonce:     big.NewInt(0),
				InputHash: big.NewInt(0),
			}
		}
	}

	// Compute Merkle root
	root := computeMerkleRoot(circuit.Transactions[:])
	log.Printf("DEBUG: Go Computed Root: %s", root.String())
	circuit.BatchRoot = root
	circuit.OldStateRoot = parseBigInt(batch.PrevStateRoot)
	circuit.NewStateRoot = parseBigInt(batch.NewStateRoot)
	circuit.BatchNumber = big.NewInt(int64(batch.Number))

	// Create witness
	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	if err != nil {
		return nil, nil, err
	}

	publicWitness, err := witness.Public()
	if err != nil {
		return nil, nil, err
	}

	// Generate proof
	proof, err := groth16.Prove(ccs, pk, witness)
	if err != nil {
		return nil, nil, err
	}

	return proof, publicWitness, nil
}

func submitProof(client *ethclient.Client, auth *bind.TransactOpts, contractAddr string, batch *Batch, proof groth16.Proof, publicWitness witness.Witness) (string, error) {
	// 1. Reconstruct batchData string
	batchDataStr := fmt.Sprintf("%d:%s:%s:%d",
		batch.Number,
		batch.PrevStateRoot,
		batch.NewStateRoot,
		batch.Timestamp)
	for _, tx := range batch.Transactions {
		batchDataStr += ":" + tx.Hash
	}
	batchData := []byte(batchDataStr)

	// 3. Extract proof points
	bn254Proof, ok := proof.(*bn254.Proof)
	if !ok {
		return "", fmt.Errorf("invalid proof type")
	}

	proofA := [2]*big.Int{bn254Proof.Ar.X.BigInt(new(big.Int)), bn254Proof.Ar.Y.BigInt(new(big.Int))}
	proofB := [2][2]*big.Int{
		{bn254Proof.Bs.X.A1.BigInt(new(big.Int)), bn254Proof.Bs.X.A0.BigInt(new(big.Int))},
		{bn254Proof.Bs.Y.A1.BigInt(new(big.Int)), bn254Proof.Bs.Y.A0.BigInt(new(big.Int))},
	}
	proofC := [2]*big.Int{bn254Proof.Krs.X.BigInt(new(big.Int)), bn254Proof.Krs.Y.BigInt(new(big.Int))}

	// 4. Recompute BatchRoot for public inputs
	var transactions [circuits.BatchSize]circuits.Transaction
	for i := 0; i < circuits.BatchSize; i++ {
		if i < len(batch.Transactions) {
			tx := batch.Transactions[i]
			transactions[i] = circuits.Transaction{
				From:      hashAddress(tx.From),
				To:        hashAddress(tx.To),
				Amount:    parseBigInt(string(tx.Value)),
				Nonce:     parseBigInt(string(tx.Nonce)),
				InputHash: hashData(tx.Data),
			}
		} else {
			transactions[i] = circuits.Transaction{
				From:      big.NewInt(0),
				To:        big.NewInt(0),
				Amount:    big.NewInt(0),
				Nonce:     big.NewInt(0),
				InputHash: big.NewInt(0),
			}
		}
	}
	batchRoot := computeMerkleRoot(transactions[:])

	// 2. Prepare arguments
	// IMPORTANT: We must use the MiMC Merkle Root as the batchHash because that is what the ZK proof verifies.
	batchHash := common.BytesToHash(BigIntTo32Bytes(batchRoot))
	newStateRoot := common.HexToHash(batch.NewStateRoot)

	publicInputs := [4]*big.Int{
		batchRoot,
		parseBigInt(batch.PrevStateRoot),
		parseBigInt(batch.NewStateRoot),
		new(big.Int).SetUint64(batch.Number),
	}

	// 5. Pack ABI and send transaction
	const abiJSON = `[{"inputs":[{"internalType":"bytes32","name":"batchHash","type":"bytes32"},{"internalType":"bytes32","name":"newStateRoot","type":"bytes32"},{"internalType":"bytes","name":"batchData","type":"bytes"},{"internalType":"uint256[2]","name":"proofA","type":"uint256[2]"},{"internalType":"uint256[2][2]","name":"proofB","type":"uint256[2][2]"},{"internalType":"uint256[2]","name":"proofC","type":"uint256[2]"},{"internalType":"uint256[4]","name":"publicInputs","type":"uint256[4]"}],"name":"registerBatch","outputs":[{"internalType":"uint256","name":"batchNumber","type":"uint256"}],"stateMutability":"nonpayable","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Debug logs
	log.Printf("DEBUG: BatchRoot (BigInt): %s\n", batchRoot.String())
	log.Printf("DEBUG: BatchHash (Hex): %s\n", batchHash.Hex())
	log.Printf("DEBUG: PublicInputs[0] (BigInt): %s\n", publicInputs[0].String())

	contract := bind.NewBoundContract(common.HexToAddress(contractAddr), parsedABI, client, client, client)
	tx, err := contract.Transact(auth, "registerBatch", batchHash, newStateRoot, batchData, proofA, proofB, proofC, publicInputs)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	return tx.Hash().Hex(), nil
}

// Helper functions
func hashAddress(addr string) *big.Int {
	// Convert hex address to big.Int
	if addr == "" {
		return big.NewInt(0)
	}
	return new(big.Int).SetBytes(common.HexToAddress(addr).Bytes())
}

func parseBigInt(s string) *big.Int {
	val, ok := new(big.Int).SetString(s, 0)
	if !ok {
		return big.NewInt(0)
	}
	return val
}

func hashData(data string) *big.Int {
	if data == "" || data == "0x" {
		return big.NewInt(0)
	}
	hash := crypto.Keccak256Hash([]byte(data))
	return new(big.Int).SetBytes(hash.Bytes())
}

func computeMerkleRoot(transactions []circuits.Transaction) *big.Int {
	h := mimc.NewMiMC()

	// Compute leaf hashes
	leaves := make([]*big.Int, circuits.BatchSize)
	for i := 0; i < circuits.BatchSize; i++ {
		h.Reset()
		h.Write(BigIntTo32Bytes(transactions[i].From.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].To.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Amount.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Nonce.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].InputHash.(*big.Int)))
		leaves[i] = new(big.Int).SetBytes(h.Sum(nil))

		if i == 0 {
			log.Printf("DEBUG: Go Leaf 0: %s", leaves[i].String())
			log.Printf("DEBUG: Go Tx 0 InputHash: %s", transactions[i].InputHash.(*big.Int).String())
		}
	}

	// Build Merkle tree
	currentLayer := leaves
	for len(currentLayer) > 1 {
		nextLayer := make([]*big.Int, len(currentLayer)/2)
		for i := 0; i < len(currentLayer); i += 2 {
			h.Reset()
			h.Write(BigIntTo32Bytes(currentLayer[i]))
			h.Write(BigIntTo32Bytes(currentLayer[i+1]))
			nextLayer[i/2] = new(big.Int).SetBytes(h.Sum(nil))
		}
		currentLayer = nextLayer
	}

	return currentLayer[0]
}

func BigIntTo32Bytes(i *big.Int) []byte {
	b := i.Bytes()
	if len(b) == 32 {
		return b
	}
	// If larger than 32 bytes (unlikely for BN254 scalar field elements), truncate
	if len(b) > 32 {
		return b[len(b)-32:]
	}
	// Pad with leading zeros
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return padded
}
