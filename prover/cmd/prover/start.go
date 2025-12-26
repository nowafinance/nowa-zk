package main

import (
	"bytes"
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
	"github.com/tannetwork/zk-sequencer/prover/internal/api"
	"github.com/tannetwork/zk-sequencer/prover/internal/storage"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start prover service to fetch batches and submit proofs",
	Long:  `Fetches transaction batches from sequencer API, generates ZK proofs, and submits to smart contract.`,
	Run:   start,
}

var (
	keysDir            string
	sequencerURL       string
	rpcURL             string
	contractAddr       string
	privateKeyHex      string
	pollInterval       int
	configPath         string
	enableParanoidMode bool
	maxRebuildAttempts int
	clearHalt          bool
	testFailure        bool
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

	// Paranoid mode flags
	startCmd.Flags().BoolVar(&enableParanoidMode, "paranoid-mode", true, "Enable proof rebuild on verification failure")
	startCmd.Flags().IntVar(&maxRebuildAttempts, "max-rebuilds", 1, "Maximum proof rebuild attempts")
	startCmd.Flags().BoolVar(&clearHalt, "clear-halt", false, "Clear halt state and resume processing")

	// Testing flags
	startCmd.Flags().BoolVar(&testFailure, "test-failure", false, "[TESTING] Intentionally corrupt proof to test error handling")
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
		if envRPCProver := os.Getenv("RPC_PROVER"); envRPCProver != "" {
			rpcURL = envRPCProver
		} else if envRPC := os.Getenv("RPC"); envRPC != "" {
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
		// Priority 1: Check .tan-zk/deployments.json in CURRENT directory (where make deploy saves it)
		localDeployPath := ".tan-zk/deployments.json"
		if data, err := os.ReadFile(localDeployPath); err == nil {
			var deployments map[string]string
			if err := json.Unmarshal(data, &deployments); err == nil {
				if addr, ok := deployments["BatchRegistry"]; ok {
					contractAddr = addr
					log.Printf("ℹ️  Auto-loaded Contract: %s (from local .tan-zk)", contractAddr)
				}
			}
		}

		// Priority 2: Check ~/.tan-zk/deployments.json (Global/Home directory)
		if contractAddr == "" {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				deploymentsPath := homeDir + "/.tan-zk/deployments.json"
				if data, err := os.ReadFile(deploymentsPath); err == nil {
					// Parse JSON to map
					var deployments map[string]string
					if err := json.Unmarshal(data, &deployments); err == nil {
						if addr, ok := deployments["BatchRegistry"]; ok {
							contractAddr = addr
							log.Printf("ℹ️  Auto-loaded Contract: %s (from home dir)", contractAddr)
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

	// Test mode warning
	if testFailure {
		log.Println()
		log.Println("⚠️  ⚠️  ⚠️  WARNING: TEST FAILURE MODE ENABLED ⚠️  ⚠️  ⚠️")
		log.Println("Proofs will be intentionally corrupted to test error handling!")
		log.Println("This mode should ONLY be used for testing.")
		log.Println()
	}
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

	// Check if --clear-halt flag was provided
	if clearHalt {
		if err := store.ClearHaltState(); err != nil {
			log.Printf("⚠️  Failed to clear halt state: %v", err)
		} else {
			log.Println("✅ Halt state cleared. Prover will resume normal operation.")
		}
	}

	// Check if prover was previously halted
	halted, reason, err := store.GetHaltState()
	if err != nil {
		log.Printf("⚠️  Failed to check halt state: %v", err)
	} else if halted {
		log.Println("========================================")
		log.Println("❌ PROVER IS HALTED")
		log.Println("========================================")
		log.Printf("Reason: %s\n", reason)
		log.Println()
		log.Println("Troubleshooting Steps:")
		log.Println("1. Review failure logs in ~/.tan-zk/prover/data/")
		log.Println("2. Check verification failure details in storage")
		log.Println("3. Verify circuit constraints match contract verifier")
		log.Println("4. After fixing, restart with --clear-halt flag:")
		log.Printf("   ./build/prover-bin start --keys-dir %s --clear-halt\n", keysDir)
		log.Println()
		log.Println("For support, visit: https://github.com/tannetwork/tan-zk/issues")
		log.Println("========================================")
		return
	}

	// Start API Server
	// Start API server
	apiServer := api.NewAPIServer(batchRegistry, store, 8081)
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Fatalf("❌ Failed to start API server: %v", err)
		}
	}()

	// Load last processed batch
	lastProcessedBatch, err := store.GetLastProcessedBatch()
	if err != nil {
		log.Printf("⚠️  Failed to load last processed batch: %v", err)
	} else if lastProcessedBatch > 0 {
		log.Printf("🔄 Resuming from batch #%d", lastProcessedBatch)
	}

	// Load last known state root from DB
	localStateRootStr, err := store.GetLastStateRoot()
	var localStateRoot *big.Int
	if err == nil && localStateRootStr != "" {
		localStateRoot = parseBigInt(localStateRootStr)
		log.Printf("📦 Loaded last known state root from DB: %s", localStateRoot.String())
	}

	// Sync with contract state to avoid re-submitting finalized batches
	onChainBatches, err := getTotalBatches(client, contractAddr)
	if err != nil {
		log.Printf("⚠️  Failed to fetch on-chain batch count: %v", err)
	} else {
		onChainBatchesU64 := onChainBatches.Uint64()
		// Always sync if local state is less than OR EQUAL to contract state
		// Equal case handles crash-consistency (saved batch num but stale state root)
		if onChainBatchesU64 >= lastProcessedBatch {
			if onChainBatchesU64 > lastProcessedBatch {
				log.Printf("⚠️  Local state is behind contract. Fast-forwarding %d -> %d", lastProcessedBatch, onChainBatchesU64)
				lastProcessedBatch = onChainBatchesU64
				if err := store.SaveLastProcessedBatch(lastProcessedBatch); err != nil {
					log.Printf("⚠️  Failed to save synced state: %v", err)
				}
			} else {
				log.Printf("ℹ️  Local batch matches contract (%d). Verifying state root consistency...", lastProcessedBatch)
			}

			// Sync localStateRoot to match the contract's current state root
			contractStateRoot, err := getCurrentStateRoot(client, contractAddr)
			if err != nil {
				log.Printf("❌ Failed to sync state root from contract: %v", err)
			} else {
				localStateRoot = contractStateRoot
				if err := store.SaveLastStateRoot(localStateRoot.String()); err != nil {
					log.Printf("⚠️  Failed to save synced state root: %v", err)
				}
				log.Printf("🔄 Synced Local State Root to: %s", localStateRoot.String())
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

		// Use paranoid mode submission (handles proof generation, verification, submission, and rebuild)
		tx, _, calculatedNewStateRoot, err := submitProofWithParanoidMode(
			client, auth, contractAddr, batch, ccs, pk, vk, localStateRoot, sequencerAddr,
			store, enableParanoidMode, maxRebuildAttempts, testFailure,
		)

		if err != nil {
			// Check if this is a halt error
			if strings.HasPrefix(err.Error(), "HALTED:") {
				// Prover has been halted - exit the main loop
				log.Println()
				log.Println("Exiting prover due to halt state...")
				return
			}

			// Other errors - log and retry after interval
			log.Printf("   ❌ Failed to process batch #%d: %v\n", batch.Number, err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		log.Printf("   ✅ Batch #%d successfully processed!\n", batch.Number)
		log.Println()

		// Save state to store
		if err := store.SaveLastProcessedBatch(batch.Number); err != nil {
			log.Printf("⚠️  Failed to save last processed batch: %v", err)
		}

		// Save proof data with tx hash (no witness needed, too large)
		txHash := ""
		if tx != nil {
			txHash = tx.Hash().Hex()
		}

		// Extract L2 transaction hashes
		txHashes := make([]string, len(batch.Transactions))
		for i, tr := range batch.Transactions {
			txHashes[i] = tr.Hash
		}

		// Save batch metadata (L1 tx hash + L2 tx hashes)
		if err := store.SaveMetadata(batch.Number, txHash, txHashes); err != nil {
			log.Printf("⚠️  Failed to save batch metadata: %v", err)
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
			nonce := parseBigInt(string(tx.Nonce))
			amount := parseBigInt(string(tx.Value))
			gasPrice := big.NewInt(1000000000) // 1 Gwei default
			gasLimit := big.NewInt(21000)      // Standard transfer gas limit

			// Debug: log first few transactions
			if i < 3 || i == len(batch.Transactions)-1 {
				log.Printf("   DEBUG TX %d: Nonce=%s, Amount=%s, From=%s, To=%s",
					i, nonce.String(), amount.String(), tx.From[:10]+"...", tx.To[:10]+"...")
			}

			circuit.Transactions[i] = circuits.Transaction{
				Nonce:    nonce,
				From:     hashAddress(tx.From),
				To:       hashAddress(tx.To),
				Amount:   amount,
				GasPrice: gasPrice,
				GasLimit: gasLimit,
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

// ErrorType represents the type of error encountered
type ErrorType int

const (
	ErrorTypeNetwork ErrorType = iota
	ErrorTypeVerification
	ErrorTypeUnknown
)

// classifyError determines the type of error from submission/transaction
func classifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	errMsg := err.Error()

	// Network/RPC errors
	if strings.Contains(errMsg, "connection") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "network") ||
		strings.Contains(errMsg, "dial") ||
		strings.Contains(errMsg, "EOF") {
		return ErrorTypeNetwork
	}

	// Verification errors (contract reverts)
	if strings.Contains(errMsg, "execution reverted") ||
		strings.Contains(errMsg, "Invalid proof") ||
		strings.Contains(errMsg, "verification failed") ||
		strings.Contains(errMsg, "verifier") {
		return ErrorTypeVerification
	}

	return ErrorTypeUnknown
}

// saveFailureData saves comprehensive failure data for debugging
func saveFailureData(batch *Batch, proof groth16.Proof, publicWitness witness.Witness, errMsg string) error {
	failureDir := os.ExpandEnv("$HOME/.tan-zk/prover/failures")
	if err := os.MkdirAll(failureDir, 0755); err != nil {
		return fmt.Errorf("failed to create failure directory: %w", err)
	}

	// Save batch data
	batchFile := fmt.Sprintf("%s/batch_%d.json", failureDir, batch.Number)
	batchData, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal batch data: %w", err)
	}
	if err := os.WriteFile(batchFile, batchData, 0644); err != nil {
		return fmt.Errorf("failed to write batch file: %w", err)
	}

	// Save proof
	proofFile := fmt.Sprintf("%s/batch_%d_proof.bin", failureDir, batch.Number)
	proofF, err := os.Create(proofFile)
	if err != nil {
		return fmt.Errorf("failed to create proof file: %w", err)
	}
	defer proofF.Close()
	if _, err := proof.WriteTo(proofF); err != nil {
		return fmt.Errorf("failed to write proof: %w", err)
	}

	// Save public witness
	witnessFile := fmt.Sprintf("%s/batch_%d_witness.bin", failureDir, batch.Number)
	witnessF, err := os.Create(witnessFile)
	if err != nil {
		return fmt.Errorf("failed to create witness file: %w", err)
	}
	defer witnessF.Close()
	if _, err := publicWitness.WriteTo(witnessF); err != nil {
		return fmt.Errorf("failed to write witness: %w", err)
	}

	// Save error log
	errorFile := fmt.Sprintf("%s/batch_%d_error.log", failureDir, batch.Number)
	errorLog := fmt.Sprintf("Batch Number: %d\nTimestamp: %s\nError: %s\n",
		batch.Number, time.Now().Format(time.RFC3339), errMsg)
	if err := os.WriteFile(errorFile, []byte(errorLog), 0644); err != nil {
		return fmt.Errorf("failed to write error log: %w", err)
	}

	log.Printf("   💾 Failure data saved to: %s/batch_%d_*", failureDir, batch.Number)
	return nil
}

// haltProver halts the prover with detailed error information
func haltProver(store *storage.ProverStore, batch *Batch, errMsg string) {
	log.Println()
	log.Println("========================================")
	log.Println("⚠️  PARANOID MODE: PROOF VERIFICATION FAILED AFTER REBUILD!")
	log.Println("❌ HALTING PROVER - MANUAL INTERVENTION REQUIRED")
	log.Println("========================================")
	log.Printf("Batch Number: %d\n", batch.Number)
	log.Printf("Error: %s\n", errMsg)
	log.Printf("Timestamp: %s\n", time.Now().Format(time.RFC3339))
	log.Println()
	log.Println("Troubleshooting Steps:")
	log.Println("1. Review failure data in: ~/.tan-zk/prover/failures/")
	log.Printf("   - Batch data: batch_%d.json\n", batch.Number)
	log.Printf("   - Proof: batch_%d_proof.bin\n", batch.Number)
	log.Printf("   - Witness: batch_%d_witness.bin\n", batch.Number)
	log.Printf("   - Error log: batch_%d_error.log\n", batch.Number)
	log.Println("2. Verify circuit constraints match contract verifier")
	log.Println("3. Check for non-deterministic proof generation bugs")
	log.Println("4. Ensure state synchronization is correct")
	log.Println("5. After fixing, restart with --clear-halt flag:")
	log.Println("   make run-prover CLEAR_HALT=true")
	log.Println("   OR: ./build/prover-bin start --keys-dir ~/.tan-zk/keys --clear-halt")
	log.Println()
	log.Println("For support, visit: https://github.com/tannetwork/tan-zk/issues")
	log.Println("========================================")

	// Set halt state in storage
	haltReason := fmt.Sprintf("Batch #%d verification failed after rebuild: %s", batch.Number, errMsg)
	if err := store.SetHaltState(haltReason); err != nil {
		log.Printf("⚠️  Failed to save halt state: %v", err)
	}
}

// submitProofWithParanoidMode handles proof submission with two-level retry and rebuild
func submitProofWithParanoidMode(
	client *ethclient.Client,
	auth *bind.TransactOpts,
	contractAddr string,
	batch *Batch,
	ccs constraint.ConstraintSystem,
	pk groth16.ProvingKey,
	vk groth16.VerifyingKey,
	localStateRoot *big.Int,
	sequencerAddr *big.Int,
	store *storage.ProverStore,
	enableParanoid bool,
	maxRebuilds int,
	testMode bool,
) (*types.Transaction, *types.Receipt, *big.Int, error) {

	// Level 1: Try with initial proof (3 quick retries for network issues)
	proof, publicWitness, calculatedNewStateRoot, err := generateProof(batch, ccs, pk, localStateRoot, sequencerAddr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	log.Println("   ✅ Proof generated")
	log.Printf("   🧮 Calculated New State Root: %s\n", calculatedNewStateRoot.String())

	// Verify locally
	log.Println("   🔍 Verifying proof locally...")
	if err := verifyLocal(proof, vk, publicWitness); err != nil {
		return nil, nil, nil, fmt.Errorf("local verification failed: %w", err)
	}
	log.Println("   ✅ Proof verified locally")

	// Attempt submission with retries
	tx, receipt, err := attemptSubmission(client, auth, contractAddr, batch, proof, localStateRoot, calculatedNewStateRoot, 3, testMode)
	if err == nil {
		return tx, receipt, calculatedNewStateRoot, nil
	}

	// Check error type
	errType := classifyError(err)
	log.Printf("   ⚠️  Submission failed: %v (Type: %v)\n", err, errType)

	// If it's a network error, don't rebuild - just fail
	if errType == ErrorTypeNetwork {
		return nil, nil, nil, fmt.Errorf("network error after retries: %w", err)
	}

	// Level 2: Paranoid mode - rebuild proof if enabled
	if !enableParanoid || errType != ErrorTypeVerification {
		return nil, nil, nil, err
	}

	log.Println()
	log.Println("   🔄 PARANOID MODE: Rebuilding proof due to verification failure...")

	for rebuild := 1; rebuild <= maxRebuilds; rebuild++ {
		log.Printf("   🔧 Rebuild attempt %d/%d\n", rebuild, maxRebuilds)

		// Regenerate proof
		proof, publicWitness, calculatedNewStateRoot, err = generateProof(batch, ccs, pk, localStateRoot, sequencerAddr)
		if err != nil {
			log.Printf("   ❌ Proof regeneration failed: %v\n", err)
			continue
		}

		log.Println("   ✅ Proof regenerated")

		// Verify locally again
		if err := verifyLocal(proof, vk, publicWitness); err != nil {
			log.Printf("   ❌ Local verification failed: %v\n", err)
			continue
		}
		log.Println("   ✅ Proof verified locally")

		// Try submission again
		tx, receipt, err = attemptSubmission(client, auth, contractAddr, batch, proof, localStateRoot, calculatedNewStateRoot, 2, testMode)
		if err == nil {
			log.Println("   ✅ Proof accepted after rebuild!")
			return tx, receipt, calculatedNewStateRoot, nil
		}

		log.Printf("   ❌ Rebuilt proof also failed: %v\n", err)
	}

	// Level 3: HALT - All attempts failed
	errMsg := fmt.Sprintf("Verification failed after %d rebuild attempts: %v", maxRebuilds, err)

	// Save failure data
	if saveErr := saveFailureData(batch, proof, publicWitness, errMsg); saveErr != nil {
		log.Printf("⚠️  Failed to save failure data: %v\n", saveErr)
	}

	// Save to storage
	var proofBuf bytes.Buffer
	proof.WriteTo(&proofBuf)
	if saveErr := store.SaveVerificationFailure(batch.Number, errMsg, proofBuf.Bytes()); saveErr != nil {
		log.Printf("⚠️  Failed to save failure to storage: %v\n", saveErr)
	}

	// Halt the prover
	haltProver(store, batch, errMsg)

	return nil, nil, nil, fmt.Errorf("HALTED: %s", errMsg)
}

// attemptSubmission tries to submit and verify a proof with retries
func attemptSubmission(
	client *ethclient.Client,
	auth *bind.TransactOpts,
	contractAddr string,
	batch *Batch,
	proof groth16.Proof,
	prevStateRoot *big.Int,
	newStateRoot *big.Int,
	maxAttempts int,
	testMode bool,
) (*types.Transaction, *types.Receipt, error) {

	var tx *types.Transaction
	var err error

	// TEST MODE: Corrupt proof to simulate verification failure
	proofToSubmit := proof
	if testMode {
		log.Println("   🧪 TEST MODE: Corrupting proof to simulate verification failure...")
		// Corrupt the proof by modifying one of its points
		bn254Proof, ok := proof.(*bn254.Proof)
		if ok {
			// Create a copy and corrupt it
			corruptedProof := *bn254Proof
			// Modify the proof by adding 1 to the X coordinate of point A
			corruptedProof.Ar.X.Add(&corruptedProof.Ar.X, &corruptedProof.Ar.X)
			proofToSubmit = &corruptedProof
		}
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		tx, err = submitProof(client, auth, contractAddr, batch, proofToSubmit, prevStateRoot, newStateRoot)
		if err != nil {
			if attempt < maxAttempts {
				log.Printf("   ⚠️  Submission attempt %d/%d failed: %v. Retrying in 5s...\n", attempt, maxAttempts, err)
				time.Sleep(5 * time.Second)
				continue
			}
			return nil, nil, fmt.Errorf("submission failed after %d attempts: %w", maxAttempts, err)
		}

		log.Printf("   ✅ Proof submitted. TxHash: %s\n", tx.Hash().Hex())
		log.Println("   ⏳ Waiting for transaction to be mined...")

		receipt, err := bind.WaitMined(context.Background(), client, tx)
		if err != nil {
			if attempt < maxAttempts {
				log.Printf("   ⚠️  Mining attempt %d/%d failed: %v. Retrying...\n", attempt, maxAttempts, err)
				time.Sleep(5 * time.Second)
				continue
			}
			return nil, nil, fmt.Errorf("transaction mining failed after %d attempts: %w", maxAttempts, err)
		}

		if receipt.Status != types.ReceiptStatusSuccessful {
			errMsg := fmt.Sprintf("transaction reverted with status: %v", receipt.Status)
			if attempt < maxAttempts {
				log.Printf("   ⚠️  Verification attempt %d/%d failed: %s. Retrying...\n", attempt, maxAttempts, errMsg)
				time.Sleep(5 * time.Second)
				continue
			}
			return tx, receipt, fmt.Errorf("execution reverted: %s", errMsg)
		}

		log.Printf("   ✅ Transaction mined. Block Number: %v\n", receipt.BlockNumber)
		return tx, receipt, nil
	}

	return nil, nil, fmt.Errorf("submission failed after %d attempts", maxAttempts)
}

// API Server Implementation
