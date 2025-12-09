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
	"github.com/consensys/gnark-crypto/ecc/bls12-377/fr"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark/backend/groth16"
	bn254 "github.com/consensys/gnark/backend/groth16/bn254"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/tannetwork/zk-sequencer/prover/bindings"
	"github.com/tannetwork/zk-sequencer/prover/circuits"
	"github.com/tannetwork/zk-sequencer/prover/internal/storage"
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
	startCmd.Flags().StringVarP(&rpcURL, "rpc-url", "r", "", "Ethereum RPC URL")

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
	// Load .env file
	_ = godotenv.Load()

	// Set defaults from env if not provided via flags
	if rpcURL == "" {
		if envRPC := os.Getenv("RPC"); envRPC != "" {
			rpcURL = envRPC
		}
	}

	if privateKeyHex == "" {
		if envKey := os.Getenv("PRIVATE_KEY"); envKey != "" {
			privateKeyHex = envKey
		}
	}

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
	ccs, pk, vk, err := loadCircuitAndKeys()
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

	// Initialize BatchRegistry contract
	batchRegistry, err := bindings.NewBatchRegistry(common.HexToAddress(contractAddr), client)
	if err != nil {
		log.Fatalf("❌ Failed to instantiate BatchRegistry contract: %v", err)
	}

	// Start API Server
	go startAPIServer(batchRegistry)

	// Initialize storage
	storePath := ".tan-zk/prover/data"
	if err := os.MkdirAll(storePath, 0755); err != nil {
		log.Fatalf("❌ Failed to create storage directory: %v", err)
	}

	store, err := storage.NewProverStore(storePath)
	if err != nil {
		log.Fatalf("❌ Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Load last processed batch
	lastProcessedBatch, err := store.GetLastProcessedBatch()
	if err != nil {
		log.Printf("⚠️  Failed to load last processed batch: %v", err)
	} else if lastProcessedBatch > 0 {
		log.Printf("🔄 Resuming from batch #%d", lastProcessedBatch)
	}

	// Sync with contract state to avoid re-submitting finalized batches
	onChainBatches, err := getTotalBatches(client, contractAddr)
	if err != nil {
		log.Printf("⚠️  Failed to fetch on-chain batch count: %v", err)
	} else {
		onChainBatchesU64 := onChainBatches.Uint64()
		if onChainBatchesU64 > lastProcessedBatch {
			log.Printf("⚠️  Local state is behind contract. Fast-forwarding %d -> %d", lastProcessedBatch, onChainBatchesU64)
			lastProcessedBatch = onChainBatchesU64
			if err := store.SaveLastProcessedBatch(lastProcessedBatch); err != nil {
				log.Printf("⚠️  Failed to save synced state: %v", err)
			}
		} else if onChainBatchesU64 < lastProcessedBatch {
			// This indicates a potential reorg or contract redeploy with old state
			log.Printf("⚠️  WARNING: Contract has FEWER batches (%d) than local state (%d). Contract might have been reset.", onChainBatchesU64, lastProcessedBatch)
			// We could reset local state here, but let's just warn for now.
			// If we really want to recover, we should probably respect the contract?
			// If contract is 0, we should probably start from 0.
			if onChainBatchesU64 == 0 {
				log.Printf("⚠️  Contract is empty. Resetting local progress to 0.")
				lastProcessedBatch = 0
			}
		}
	}

	// Load last known state root from DB
	localStateRootStr, err := store.GetLastStateRoot()
	var localStateRoot *big.Int
	if err == nil && localStateRootStr != "" {
		localStateRoot = parseBigInt(localStateRootStr)
		log.Printf("📦 Loaded last known state root from DB: %s", localStateRoot.String())
	}

	// Main prover loop
	log.Println("🚀 Starting prover loop...")
	log.Println("   Polling for new batches...")
	log.Println()

	for {
		// 1. Get the latest batch number from the sequencer
		latestBatch, err := fetchLatestBatch(sequencerURL)
		if err != nil {
			log.Printf("⚠️  Failed to fetch latest batch info: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		if latestBatch == nil {
			log.Println("⏳ No batches generated yet. Waiting...")
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		log.Printf("✅ Fetched latest batch info: %v\n", latestBatch.Number)

		// 2. Determine the next batch we want to process
		nextBatchNum := lastProcessedBatch + 1

		// 3. Check if we are strictly behind the latest batch (Process up to N-1)
		if nextBatchNum >= latestBatch.Number {
			log.Printf("Waiting for new batches... (Latest: %d, Processed: %d, Target: < %d)\n", latestBatch.Number, lastProcessedBatch, latestBatch.Number)
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

		// 5. Ensure we have a valid OldStateRoot
		// If we don't have a local one (startup), fetch from contract
		if localStateRoot == nil {
			contractStateRoot, err := getCurrentStateRoot(client, contractAddr)
			if err != nil {
				log.Printf("⚠️  Failed to fetch contract state root: %v\n", err)
				time.Sleep(time.Duration(pollInterval) * time.Second)
				continue
			}
			localStateRoot = contractStateRoot
			log.Printf("   🏛️  Initialized State Root from Contract: %s\n", localStateRoot.String())
		} else {
			log.Printf("   💾 Using Local OldStateRoot: %s\n", localStateRoot.String())
		}

		// Generate proof using LOCAL state root as PrevStateRoot
		log.Println("   🔐 Generating proof...")
		sequencerAddr := hashAddress(auth.From.Hex())
		proof, publicWitness, calculatedNewStateRoot, err := generateProof(batch, ccs, pk, localStateRoot, sequencerAddr)
		if err != nil {
			log.Printf("   ❌ Failed to generate proof: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		log.Println("   ✅ Proof generated")
		log.Printf("   🧮 Calculated New State Root: %s\n", calculatedNewStateRoot.String())

		// Verify locally
		log.Println("   🔍 Verifying proof locally...")
		if err := verifyLocal(proof, vk, publicWitness); err != nil {
			log.Printf("   ❌ Local verification failed: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		log.Println("   ✅ Proof verified locally")

		// Submit proof to contract
		// Submit proof to contract with retry logic
		log.Println("   📡 Submitting proof to contract...")

		var tx *types.Transaction
		submitted := false

		for attempt := 1; attempt <= 3; attempt++ {
			tx, err = submitProof(client, auth, contractAddr, batch, proof, localStateRoot, calculatedNewStateRoot)
			if err == nil {
				submitted = true
				break
			}

			if attempt < 3 {
				log.Printf("   ⚠️  Submission failed: %v. Retrying in 5s... (Attempt %d/3)\n", err, attempt)
				time.Sleep(5 * time.Second)
			}
		}

		if !submitted {
			log.Printf("   ❌ Failed to submit proof after 3 attempts: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		log.Printf("   ✅ Proof submitted. TxHash: %s\n", tx.Hash().Hex())
		log.Println("   ⏳ Waiting for transaction to be mined...")

		receipt, err := bind.WaitMined(context.Background(), client, tx)
		if err != nil {
			log.Printf("   ❌ Transaction mining failed: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		if receipt.Status != types.ReceiptStatusSuccessful {
			log.Printf("   ❌ Transaction reverted with status: %v\n", receipt.Status)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		log.Printf("   ✅ Transaction mined. Block Number: %v\n", receipt.BlockNumber)
		log.Println()

		// Save state to store
		if err := store.SaveLastProcessedBatch(batch.Number); err != nil {
			log.Printf("⚠️  Failed to save last processed batch: %v", err)
		}
		if err := store.SaveProof(batch.Number, proof, publicWitness); err != nil {
			log.Printf("⚠️  Failed to save proof data: %v", err)
		}

		// Update local state root and persist
		localStateRoot = calculatedNewStateRoot
		if err := store.SaveLastStateRoot(localStateRoot.String()); err != nil {
			log.Printf("⚠️  Failed to persist state root: %v", err)
		}
		log.Printf("   💾 Updated Local State Root to: %s\n", localStateRoot.String())

		lastProcessedBatch = batch.Number
	}
}

func verifyLocal(proof groth16.Proof, vk groth16.VerifyingKey, publicWitness witness.Witness) error {
	return groth16.Verify(proof, vk, publicWitness)
}

func loadCircuitAndKeys() (constraint.ConstraintSystem, groth16.ProvingKey, groth16.VerifyingKey, error) {
	// Load compiled circuit
	ccsFile, err := os.Open(keysDir + "/rollup.r1cs")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open circuit file: %w", err)
	}
	defer ccsFile.Close()

	ccs := groth16.NewCS(ecc.BN254)
	if _, err := ccs.ReadFrom(ccsFile); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read circuit: %w", err)
	}

	// Load proving key
	pkFile, err := os.Open(keysDir + "/rollup.pk")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open proving key: %w", err)
	}
	defer pkFile.Close()

	pk := groth16.NewProvingKey(ecc.BN254)
	if _, err := pk.ReadFrom(pkFile); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read proving key: %w", err)
	}

	// Load verifying key
	vkFile, err := os.Open(keysDir + "/rollup.vk")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open verifying key: %w", err)
	}
	defer vkFile.Close()

	vk := groth16.NewVerifyingKey(ecc.BN254)
	if _, err := vk.ReadFrom(vkFile); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read verifying key: %w", err)
	}

	return ccs, pk, vk, nil
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

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // No batches yet
	}

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

func generateProof(batch *Batch, ccs constraint.ConstraintSystem, pk groth16.ProvingKey, prevStateRoot *big.Int, sequencerAddr *big.Int) (groth16.Proof, witness.Witness, *big.Int, error) {
	// Convert batch to circuit witness
	var circuit circuits.Circuit

	// Fill transactions (pad with zeros if needed)
	for i := 0; i < circuits.BatchSize; i++ {
		if i < len(batch.Transactions) {
			tx := batch.Transactions[i]
			circuit.Transactions[i] = circuits.Transaction{
				Nonce:    parseBigInt(string(tx.Nonce)),
				From:     hashAddress(tx.From),
				To:       hashAddress(tx.To),
				Amount:   parseBigInt(string(tx.Value)),
				GasPrice: big.NewInt(1000000000), // 1 Gwei default
				GasLimit: big.NewInt(21000),      // Standard transfer gas limit
				Data:     hashData(tx.Data),
			}
		} else {
			// Pad with zero transactions
			circuit.Transactions[i] = circuits.Transaction{
				Nonce:    big.NewInt(0),
				From:     big.NewInt(0),
				To:       big.NewInt(0),
				Amount:   big.NewInt(0),
				GasPrice: big.NewInt(0),
				GasLimit: big.NewInt(0),
				Data:     big.NewInt(0),
			}
		}
	}

	// Compute Merkle root
	root := computeMerkleRoot(circuit.Transactions[:])
	log.Printf("DEBUG: Go Computed Root: %s", root.String())

	// Set public inputs
	circuit.BatchRoot = root
	circuit.PrevStateRoot = prevStateRoot // Use contract's state root

	// Compute NewStateRoot using circuit logic (rolling hash)
	circuit.NewStateRoot = computeRollingHash(circuit.PrevStateRoot.(*big.Int), circuit.Transactions[:])
	log.Printf("DEBUG: Go Computed State Root: %s", circuit.NewStateRoot.(*big.Int).String())

	circuit.BatchNumber = big.NewInt(int64(batch.Number))
	circuit.Timestamp = big.NewInt(batch.Timestamp)
	circuit.SequencerAddr = sequencerAddr

	// Create witness
	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	if err != nil {
		return nil, nil, nil, err
	}

	publicWitness, err := witness.Public()
	if err != nil {
		return nil, nil, nil, err
	}

	// Generate proof
	proof, err := groth16.Prove(ccs, pk, witness)
	if err != nil {
		return nil, nil, nil, err
	}

	return proof, publicWitness, circuit.NewStateRoot.(*big.Int), nil
}

func submitProof(client *ethclient.Client, auth *bind.TransactOpts, contractAddr string, batch *Batch, proof groth16.Proof, prevStateRoot *big.Int, newStateRootBig *big.Int) (*types.Transaction, error) {
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
		return nil, fmt.Errorf("invalid proof type")
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
				Nonce:    parseBigInt(string(tx.Nonce)),
				From:     hashAddress(tx.From),
				To:       hashAddress(tx.To),
				Amount:   parseBigInt(string(tx.Value)),
				GasPrice: big.NewInt(1000000000),
				GasLimit: big.NewInt(21000),
				Data:     hashData(tx.Data),
			}
		} else {
			transactions[i] = circuits.Transaction{
				Nonce:    big.NewInt(0),
				From:     big.NewInt(0),
				To:       big.NewInt(0),
				Amount:   big.NewInt(0),
				GasPrice: big.NewInt(0),
				GasLimit: big.NewInt(0),
				Data:     big.NewInt(0),
			}
		}
	}
	batchRoot := computeMerkleRoot(transactions[:])

	// 2. Prepare arguments
	// Use Keccak256 for batchHash (DA check)
	batchHash := crypto.Keccak256Hash(batchData)

	// Use calculated NewStateRoot (converted to bytes32)
	newStateRoot := common.BytesToHash(BigIntTo32Bytes(newStateRootBig))

	publicInputs := [6]*big.Int{
		batchRoot,
		prevStateRoot,
		newStateRootBig,
		new(big.Int).SetUint64(batch.Number),
		big.NewInt(batch.Timestamp),
		hashAddress(auth.From.Hex()),
	}

	// 5. Pack ABI and send transaction
	// Added getCurrentStateRoot to ABI
	const abiJSON = `[{"inputs":[{"internalType":"bytes32","name":"batchHash","type":"bytes32"},{"internalType":"bytes32","name":"newStateRoot","type":"bytes32"},{"internalType":"bytes","name":"batchData","type":"bytes"},{"internalType":"uint256[2]","name":"proofA","type":"uint256[2]"},{"internalType":"uint256[2][2]","name":"proofB","type":"uint256[2][2]"},{"internalType":"uint256[2]","name":"proofC","type":"uint256[2]"},{"internalType":"uint256[6]","name":"publicInputs","type":"uint256[6]"}],"name":"registerBatch","outputs":[{"internalType":"uint256","name":"batchNumber","type":"uint256"}],"stateMutability":"nonpayable","type":"function"}, {"inputs":[],"name":"getCurrentStateRoot","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	// Debug logs
	log.Printf("DEBUG: BatchRoot (BigInt): %s\n", batchRoot.String())
	log.Printf("DEBUG: BatchHash (Hex): %s\n", batchHash.Hex())
	log.Printf("DEBUG: PublicInputs[0] (BatchRoot): %s\n", publicInputs[0].String())
	log.Printf("DEBUG: PublicInputs[1] (OldStateRoot): %s\n", publicInputs[1].String())
	log.Printf("DEBUG: PublicInputs[2] (NewStateRoot): %s\n", publicInputs[2].String())
	log.Printf("DEBUG: PublicInputs[3] (BatchNumber): %s\n", publicInputs[3].String())
	log.Printf("DEBUG: PublicInputs[4] (Timestamp): %s\n", publicInputs[4].String())
	log.Printf("DEBUG: PublicInputs[5] (Sequencer): %s\n", publicInputs[5].String())
	log.Printf("DEBUG: Expected BatchNumber: %d\n", batch.Number)

	contract := bind.NewBoundContract(common.HexToAddress(contractAddr), parsedABI, client, client, client)
	tx, err := contract.Transact(auth, "registerBatch", batchHash, newStateRoot, batchData, proofA, proofB, proofC, publicInputs)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	return tx, nil
}

func getCurrentStateRoot(client *ethclient.Client, contractAddr string) (*big.Int, error) {
	const abiJSON = `[{"inputs":[],"name":"getCurrentStateRoot","outputs":[{"internalType":"bytes32","name":"","type":"bytes32"}],"stateMutability":"view","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	contract := bind.NewBoundContract(common.HexToAddress(contractAddr), parsedABI, client, client, client)
	var result []interface{}
	err = contract.Call(&bind.CallOpts{}, &result, "getCurrentStateRoot")
	if err != nil {
		return nil, fmt.Errorf("failed to call getCurrentStateRoot: %w", err)
	}

	// Convert bytes32 to big.Int
	rootBytes := result[0].([32]byte)
	return new(big.Int).SetBytes(rootBytes[:]), nil
}

func getTotalBatches(client *ethclient.Client, contractAddr string) (*big.Int, error) {
	const abiJSON = `[{"inputs":[],"name":"totalBatches","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	contract := bind.NewBoundContract(common.HexToAddress(contractAddr), parsedABI, client, client, client)
	var result []interface{}
	err = contract.Call(&bind.CallOpts{}, &result, "totalBatches")
	if err != nil {
		return nil, fmt.Errorf("failed to call totalBatches: %w", err)
	}

	return result[0].(*big.Int), nil
}

// Helper functions with proper field reduction

// hashAddress converts Ethereum address to BN254 field element
func hashAddress(addr string) *big.Int {
	if addr == "" {
		return big.NewInt(0)
	}

	addrBytes := common.HexToAddress(addr).Bytes()
	val := new(big.Int).SetBytes(addrBytes)

	// Reduce to BN254 scalar field
	var frVal fr.Element
	frVal.SetBigInt(val)
	return frVal.BigInt(new(big.Int))
}

// parseBigInt parses string to field element
func parseBigInt(s string) *big.Int {
	val, ok := new(big.Int).SetString(s, 0)
	if !ok {
		return big.NewInt(0)
	}

	// Reduce to BN254 scalar field
	var frVal fr.Element
	frVal.SetBigInt(val)
	return frVal.BigInt(new(big.Int))
}

// hashData converts transaction data to field element
func hashData(data string) *big.Int {
	if data == "" || data == "0x" {
		return big.NewInt(0)
	}

	// Hash the data first
	hash := crypto.Keccak256Hash([]byte(data))
	val := new(big.Int).SetBytes(hash.Bytes())

	// Reduce to BN254 scalar field
	var frVal fr.Element
	frVal.SetBigInt(val)
	return frVal.BigInt(new(big.Int))
}

// BigIntTo32Bytes converts field element to 32-byte array for MiMC
func BigIntTo32Bytes(i *big.Int) []byte {
	b := i.Bytes()
	if len(b) == 32 {
		return b
	}
	if len(b) > 32 {
		return b[len(b)-32:]
	}
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return padded
}

// computeMerkleRoot - Fixed version with proper field arithmetic
func computeMerkleRoot(transactions []circuits.Transaction) *big.Int {
	h := mimc.NewMiMC()

	// Compute leaf hashes - must match circuit's hash order
	leaves := make([]*big.Int, circuits.BatchSize)
	shift64 := new(big.Int).Lsh(big.NewInt(1), 64)
	shift128 := new(big.Int).Lsh(big.NewInt(1), 128)

	for i := 0; i < circuits.BatchSize; i++ {
		h.Reset()

		// Pack fields
		nonce := transactions[i].Nonce.(*big.Int)
		gasLimit := transactions[i].GasLimit.(*big.Int)
		gasPrice := transactions[i].GasPrice.(*big.Int)

		packed := new(big.Int).Set(nonce)
		packed.Add(packed, new(big.Int).Mul(gasLimit, shift64))
		packed.Add(packed, new(big.Int).Mul(gasPrice, shift128))

		h.Write(BigIntTo32Bytes(packed))
		h.Write(BigIntTo32Bytes(transactions[i].From.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].To.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Amount.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Data.(*big.Int)))

		leaves[i] = new(big.Int).SetBytes(h.Sum(nil))

		if i == 0 {
			log.Printf("DEBUG: Go Leaf 0: %s", leaves[i].String())
			log.Printf("DEBUG: Go Tx 0 Data: %s", transactions[i].Data.(*big.Int).String())
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

// computeRollingHash computes the state root transition using rolling hash logic
// This matches the circuit's computeStateTransition function
// computeRollingHash computes the state root transition using rolling hash logic
// This matches the circuit's computeStateTransition function
func computeRollingHash(prevStateRoot *big.Int, transactions []circuits.Transaction) *big.Int {
	h := mimc.NewMiMC()
	currentStateRoot := prevStateRoot

	shift64 := new(big.Int).Lsh(big.NewInt(1), 64)
	shift128 := new(big.Int).Lsh(big.NewInt(1), 128)

	for i := 0; i < circuits.BatchSize; i++ {
		// 1. Compute TxHash (Leaf)
		h.Reset()

		nonce := transactions[i].Nonce.(*big.Int)
		gasLimit := transactions[i].GasLimit.(*big.Int)
		gasPrice := transactions[i].GasPrice.(*big.Int)

		packed := new(big.Int).Set(nonce)
		packed.Add(packed, new(big.Int).Mul(gasLimit, shift64))
		packed.Add(packed, new(big.Int).Mul(gasPrice, shift128))

		h.Write(BigIntTo32Bytes(packed))
		h.Write(BigIntTo32Bytes(transactions[i].From.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].To.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Amount.(*big.Int)))
		h.Write(BigIntTo32Bytes(transactions[i].Data.(*big.Int)))
		txHash := h.Sum(nil)

		// 2. Update State Root: Hash(StateRoot, TxHash)
		h.Reset()
		h.Write(BigIntTo32Bytes(currentStateRoot))
		h.Write(txHash)
		currentStateRoot = new(big.Int).SetBytes(h.Sum(nil))
	}
	return currentStateRoot
}

// API Server Implementation

type BatchResponse struct {
	BatchNumber  uint64 `json:"batchNumber"`
	BatchHash    string `json:"batchHash"`
	OldStateRoot string `json:"oldStateRoot"`
	NewStateRoot string `json:"newStateRoot"`
	Submitter    string `json:"submitter"`
	Timestamp    uint64 `json:"timestamp"`
	VerifiedAt   uint64 `json:"verifiedAt"`
	Status       uint8  `json:"status"`
}

func startAPIServer(registry *bindings.BatchRegistry) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Prover API</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; line-height: 1.6; color: #333; }
        h1 { border-bottom: 2px solid #eee; padding-bottom: 10px; }
        table { width: 100%; border-collapse: collapse; margin-top: 20px; box-shadow: 0 1px 3px rgba(0,0,0,0.1); border-radius: 8px; overflow: hidden; }
        th, td { text-align: left; padding: 12px 15px; border-bottom: 1px solid #eee; }
        th { background-color: #f8f9fa; font-weight: 600; text-transform: uppercase; font-size: 0.85rem; letter-spacing: 0.5px; }
        tr:last-child td { border-bottom: none; }
        tr:hover { background-color: #f8f9fa; }
        code { background-color: #f1f3f5; padding: 2px 5px; border-radius: 4px; font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace; font-size: 0.9em; color: #d63384; }
        a { color: #228be6; text-decoration: none; }
        a:hover { text-decoration: underline; }
        .method { font-weight: bold; color: #228be6; }
    </style>
</head>
<body>
    <h1>Prover API</h1>
    <p>Available endpoints for querying batch status from the ZK Rollup Prover.</p>
    <table>
        <thead>
            <tr>
                <th>Method</th>
                <th>Endpoint</th>
                <th>Description</th>
            </tr>
        </thead>
        <tbody>
            <tr>
                <td><span class="method">GET</span></td>
                <td><a href="/batches/latest">/batches/latest</a></td>
                <td>Get details of the latest verified batch</td>
            </tr>
            <tr>
                <td><span class="method">GET</span></td>
                <td><code>/batches/{id}</code></td>
                <td>Get details of a specific batch by ID (e.g., <a href="/batches/1">/batches/1</a>)</td>
            </tr>
        </tbody>
    </table>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	mux.HandleFunc("/batches/latest", func(w http.ResponseWriter, r *http.Request) {
		totalBatches, err := registry.TotalBatches(&bind.CallOpts{})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get total batches: %v", err), http.StatusInternalServerError)
			return
		}

		if totalBatches.Cmp(big.NewInt(0)) == 0 {
			http.Error(w, "No batches found", http.StatusNotFound)
			return
		}

		batch, err := registry.GetBatch(&bind.CallOpts{}, totalBatches)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get batch: %v", err), http.StatusInternalServerError)
			return
		}

		resp := BatchResponse{
			BatchNumber:  totalBatches.Uint64(),
			BatchHash:    common.BytesToHash(batch.BatchHash[:]).Hex(),
			OldStateRoot: common.BytesToHash(batch.OldStateRoot[:]).Hex(),
			NewStateRoot: common.BytesToHash(batch.NewStateRoot[:]).Hex(),
			Submitter:    batch.Submitter.Hex(),
			Timestamp:    batch.Timestamp.Uint64(),
			VerifiedAt:   batch.VerifiedAt.Uint64(),
			Status:       batch.Status,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/batches/", func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/batches/")
		if idStr == "" {
			http.Error(w, "Batch ID required", http.StatusBadRequest)
			return
		}

		batchID, ok := new(big.Int).SetString(idStr, 10)
		if !ok {
			http.Error(w, "Invalid batch ID", http.StatusBadRequest)
			return
		}

		batch, err := registry.GetBatch(&bind.CallOpts{}, batchID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get batch: %v", err), http.StatusInternalServerError)
			return
		}

		resp := BatchResponse{
			BatchNumber:  batchID.Uint64(),
			BatchHash:    common.BytesToHash(batch.BatchHash[:]).Hex(),
			OldStateRoot: common.BytesToHash(batch.OldStateRoot[:]).Hex(),
			NewStateRoot: common.BytesToHash(batch.NewStateRoot[:]).Hex(),
			Submitter:    batch.Submitter.Hex(),
			Timestamp:    batch.Timestamp.Uint64(),
			VerifiedAt:   batch.VerifiedAt.Uint64(),
			Status:       batch.Status,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	log.Println("🌍 Starting Prover API on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatalf("❌ Failed to start API server: %v", err)
	}
}
