package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"golang.org/x/crypto/sha3"
	"strings"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bls12-377/fr"
	gomimc "github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	bn254 "github.com/consensys/gnark/backend/groth16/bn254"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/math/emulated"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/nowafinance/nowa-zk/prover/bindings"
	"github.com/nowafinance/nowa-zk/prover/circuits"
	"github.com/nowafinance/nowa-zk/prover/internal/api"
	"github.com/nowafinance/nowa-zk/prover/internal/storage"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start prover service to fetch batches and submit proofs",
	Long:  `Fetches transaction batches from indexer API, generates ZK proofs, and submits to smart contract.`,
	Run:   start,
}

var (
	keysDir            string
	indexerURL         string
	rpcURL             string
	contractAddr       string
	privateKeyHex      string
	pollInterval       int
	configPath         string
	enableParanoidMode bool
	maxRebuildAttempts int
	clearHalt          bool
	testFailure        bool
	dataDir            string
)

func init() {
	// Keys directory (default: ./keys)
	startCmd.Flags().StringVarP(&keysDir, "keys-dir", "k", "./keys", "")

	// Data directory (default: ~/.nowa-zk/prover/data)
	startCmd.Flags().StringVarP(&dataDir, "data-dir", "d", "", "Directory to store prover state database (default: ~/.nowa-zk/prover/data)")

	// Indexer API URL (default: http://localhost:8080)
	startCmd.Flags().StringVarP(&indexerURL, "indexer-url", "s", "http://localhost:8080", "")

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

// Batch represents a batch from indexer API
type Batch struct {
	Number        uint64        `json:"number"`
	Hash          string        `json:"hash"`
	PrevStateRoot string        `json:"oldStateRoot"`
	NewStateRoot  string        `json:"newStateRoot"`
	Timestamp     int64         `json:"timestamp"`
	Transactions  []Transaction `json:"transactions"`
	Trades        []ParsedTrade `json:"trades,omitempty"`
}

type ParsedTrade struct {
	MessageHash   string `json:"messageHash"`
	PubKeyX       string `json:"pubKeyX"`
	PubKeyY       string `json:"pubKeyY"`
	SigR          string `json:"sigR"`
	SigS          string `json:"sigS"`
	SignerAddress string `json:"signerAddress"`
}

// Transaction from indexer
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
		if envRPCProver := os.Getenv("L1_RPC_URL"); envRPCProver != "" {
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

	// 1. Auto-load configuration and deployments map
	var loadedDeployments map[string]string

	// Try local first
	localDeployPath := ".nowa-zk/deployments.json"
	if data, err := os.ReadFile(localDeployPath); err == nil {
		if err := json.Unmarshal(data, &loadedDeployments); err == nil {
			if contractAddr == "" {
				if addr, ok := loadedDeployments["BatchRegistry"]; ok {
					contractAddr = addr
					log.Printf("ℹ️  Auto-loaded Contract: %s (from local .nowa-zk)", contractAddr)
				}
			}
		}
	}

	// Try home directory if local didn't work for deployments map
	if loadedDeployments == nil {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			deploymentsPath := homeDir + "/.nowa-zk/deployments.json"
			if data, err := os.ReadFile(deploymentsPath); err == nil {
				if err := json.Unmarshal(data, &loadedDeployments); err == nil {
					if contractAddr == "" {
						if addr, ok := loadedDeployments["BatchRegistry"]; ok {
							contractAddr = addr
							log.Printf("ℹ️  Auto-loaded Contract: %s (from home dir)", contractAddr)
						}
					}
				}
			}
		}
	}

	if privateKeyHex == "" {
		// Try to load from .nowa-zk/secrets.env
		if data, err := os.ReadFile(".nowa-zk/secrets.env"); err == nil {
			content := string(data)
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRIVATE_KEY=") {
					privateKeyHex = strings.TrimSpace(strings.TrimPrefix(line, "PRIVATE_KEY="))
					log.Println("ℹ️  Auto-loaded Private Key from .nowa-zk/secrets.env")
					break
				}
			}
		}
	}

	// Validate required arguments
	if contractAddr == "" {
		log.Fatal("❌ Contract address required. Use --contract flag or ensure .nowa-zk/deployments.json exists.")
	}
	if privateKeyHex == "" {
		log.Fatal("❌ Private key required. Use --private-key flag or ensure .nowa-zk/secrets.env exists.")
	}
	log.Println("========================================")
	log.Println("  ZK Rollup Prover Service")
	log.Println("========================================")
	log.Printf("Indexer API: %s\n", indexerURL)
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

	// Initialize storage (use home directory for consistency with keys and deployments, unless overridden)
	var storePath string
	if dataDir != "" {
		storePath = dataDir
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("❌ Failed to get home directory: %v", err)
		}
		storePath = filepath.Join(homeDir, ".nowa-zk", "prover", "data")
	}

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
		log.Println("1. Review failure logs in ~/.nowa-zk/prover/data/")
		log.Println("2. Check verification failure details in storage")
		log.Println("3. Verify circuit constraints match contract verifier")
		log.Println("4. After fixing, restart with --clear-halt flag:")
		log.Printf("   ./build/prover-bin start --keys-dir %s --clear-halt\n", keysDir)
		log.Println()
		log.Println("For support, visit: https://github.com/nowafinance/nowa-zk/issues")
		log.Println("========================================")
		return
	}

	// Start API Server
	// Start API server
	apiServer := api.NewAPIServer(batchRegistry, store, loadedDeployments, 8081)
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

	// State root syncing has been removed as the architecture has moved to purely signature verification.

	// Main prover loop
	log.Println("🚀 Starting prover loop...")
	log.Println("   Polling for new batches...")
	log.Println()

	var consecutiveFailures int
	var failingBatch uint64

	for {
		// 1. Get the latest batch number from the indexer
		latestBatch, err := fetchLatestBatch(indexerURL)
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

		// 3. Check if we are strictly behind the latest batch (Process up to N)
		if nextBatchNum > latestBatch.Number {
			log.Printf("Waiting for new batches... (Latest: %d, Processed: %d, Target: <= %d)\n", latestBatch.Number, lastProcessedBatch, latestBatch.Number)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		// 4. Fetch the specific batch we want to process
		batch, err := fetchBatch(indexerURL, nextBatchNum)
		if err != nil {
			log.Printf("⚠️  Failed to fetch batch #%d: %v\n", nextBatchNum, err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		log.Printf("📦 Processing batch #%d (%d transactions)\n", batch.Number, len(batch.Transactions))

		// State root logic removed.

		log.Println("   🔐 Generating proofs for chunks...")
		totalTrades := len(batch.Trades)
		if totalTrades == 0 {
			log.Printf("   ℹ️  No trades in batch #%d, skipping...\n", batch.Number)
if err := store.SaveLastProcessedBatch(batch.Number); err != nil {
				log.Printf("⚠️  Failed to save last processed batch: %v", err)
			}
			lastProcessedBatch = batch.Number
			continue
		}

		numChunks := (totalTrades + circuits.TradeBatchSize - 1) / circuits.TradeBatchSize

		var finalErr error
		var lastTx *types.Transaction

		for chunkIdx := 0; chunkIdx < numChunks; chunkIdx++ {
			startIdx := chunkIdx * circuits.TradeBatchSize
			endIdx := startIdx + circuits.TradeBatchSize
			if endIdx > totalTrades {
				endIdx = totalTrades
			}
			chunkTrades := batch.Trades[startIdx:endIdx]

			log.Printf("   🧩 Processing chunk %d/%d (%d trades)\n", chunkIdx+1, numChunks, len(chunkTrades))

			tx, _, err := submitProofWithParanoidMode(
				client, auth, contractAddr, batch, chunkIdx, chunkTrades, ccs, pk, vk,
				store, enableParanoidMode, maxRebuildAttempts, testFailure,
			)

			if err != nil {
				finalErr = err
				break
			}
			if tx != nil {
				lastTx = tx
			}
		}

		if finalErr != nil {
			if strings.HasPrefix(finalErr.Error(), "HALTED:") {
				log.Println()
				log.Println("Exiting prover due to halt state...")
				return
			}
			log.Printf("   ❌ Failed to process batch #%d: %v\n", batch.Number, finalErr)

			if failingBatch == batch.Number {
				consecutiveFailures++
			} else {
				failingBatch = batch.Number
				consecutiveFailures = 1
			}

			if consecutiveFailures >= 5 {
				log.Printf("   ☠️ Poison pill batch detected! Batch #%d failed 5 times. Skipping...\n", batch.Number)
				if err := store.SaveLastProcessedBatch(batch.Number); err != nil {
					log.Printf("⚠️  Failed to save last processed batch: %v", err)
				}
				// Save metadata with status 2 (failed)
				if err := store.SaveMetadata(batch.Number, batch.Hash, "", nil, auth.From.Hex(), "0x0", 2); err != nil {
					log.Printf("⚠️  Failed to save batch metadata: %v", err)
				}
				lastProcessedBatch = batch.Number
				consecutiveFailures = 0
			} else {
				time.Sleep(time.Duration(pollInterval) * time.Second)
			}
			continue
		}

		consecutiveFailures = 0 // reset on success

		log.Printf("   ✅ Batch #%d successfully processed!\n", batch.Number)
		log.Println()

		// Save state to store
		if err := store.SaveLastProcessedBatch(batch.Number); err != nil {
			log.Printf("⚠️  Failed to save last processed batch: %v", err)
		}

		// Save proof data with tx hash (no witness needed, too large)
		txHash := ""
		if lastTx != nil {
			txHash = lastTx.Hash().Hex()
		}

		// Extract L2 transaction hashes
		txHashes := make([]string, len(batch.Transactions))
		for i, tr := range batch.Transactions {
			txHashes[i] = tr.Hash
		}

		// Save batch metadata (batch hash, L1 tx hash + L2 tx hashes)
		// Status 1 = Verified (since we just submitted proof and got txHash)
		submitter := auth.From.Hex()
		if err := store.SaveMetadata(batch.Number, batch.Hash, txHash, txHashes, submitter, "0x0", 1); err != nil {
			log.Printf("⚠️  Failed to save batch metadata: %v", err)
		}

		lastProcessedBatch = batch.Number
	}
}

func verifyLocal(proof groth16.Proof, vk groth16.VerifyingKey, publicWitness witness.Witness) error {
	return groth16.Verify(proof, vk, publicWitness, backend.WithVerifierHashToFieldFunction(sha3.NewLegacyKeccak256()))
}

func loadCircuitAndKeys() (constraint.ConstraintSystem, groth16.ProvingKey, groth16.VerifyingKey, error) {
	// Load compiled circuit
	ccsFile, err := os.Open(keysDir + "/trade.r1cs")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open circuit file: %w", err)
	}
	defer ccsFile.Close()

	ccs := groth16.NewCS(ecc.BN254)
	if _, err := ccs.ReadFrom(ccsFile); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read circuit: %w", err)
	}

	// Load proving key
	pkFile, err := os.Open(keysDir + "/trade.pk")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open proving key: %w", err)
	}
	defer pkFile.Close()

	pk := groth16.NewProvingKey(ecc.BN254)
	if _, err := pk.ReadFrom(pkFile); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read proving key: %w", err)
	}

	// Load verifying key
	vkFile, err := os.Open(keysDir + "/trade.vk")
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

func fetchLatestBatch(indexerURL string) (*Batch, error) {
	resp, err := http.Get(indexerURL + "/prover/batch/latest")
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

func fetchBatch(indexerURL string, number uint64) (*Batch, error) {
	url := fmt.Sprintf("%s/prover/batch/%d", indexerURL, number)
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

func generateProof(trades []ParsedTrade, ccs constraint.ConstraintSystem, pk groth16.ProvingKey) (groth16.Proof, witness.Witness, error) {
	var circuit circuits.BatchTradeSignatureCircuit

	if len(trades) == 0 {
		return nil, nil, fmt.Errorf("no trades in chunk to generate proof")
	}

	validHash, _ := hex.DecodeString(trades[0].MessageHash)
	validPx, _ := hex.DecodeString(trades[0].PubKeyX)
	validPy, _ := hex.DecodeString(trades[0].PubKeyY)
	validR, _ := hex.DecodeString(trades[0].SigR)
	validS, _ := hex.DecodeString(trades[0].SigS)

	for i := 0; i < circuits.TradeBatchSize; i++ {
		var hashHex, pxHex, pyHex, rHex, sHex []byte
		if i < len(trades) {
			hashHex, _ = hex.DecodeString(trades[i].MessageHash)
			pxHex, _ = hex.DecodeString(trades[i].PubKeyX)
			pyHex, _ = hex.DecodeString(trades[i].PubKeyY)
			rHex, _ = hex.DecodeString(trades[i].SigR)
			sHex, _ = hex.DecodeString(trades[i].SigS)
		} else {
			hashHex = validHash
			pxHex = validPx
			pyHex = validPy
			rHex = validR
			sHex = validS
		}

		circuit.MessageHashes[i] = emulated.ValueOf[emulated.Secp256k1Fr](hashHex)
		circuit.PubKeys[i].X = emulated.ValueOf[emulated.Secp256k1Fp](pxHex)
		circuit.PubKeys[i].Y = emulated.ValueOf[emulated.Secp256k1Fp](pyHex)
		circuit.Sigs[i].R = emulated.ValueOf[emulated.Secp256k1Fr](rHex)
		circuit.Sigs[i].S = emulated.ValueOf[emulated.Secp256k1Fr](sHex)
	}

	goMiMC := gomimc.NewMiMC()
	for i := 0; i < circuits.TradeBatchSize; i++ {
		var hashHex []byte
		if i < len(trades) {
			hashHex, _ = hex.DecodeString(trades[i].MessageHash)
		} else {
			hashHex, _ = hex.DecodeString(trades[0].MessageHash)
		}

		val := new(big.Int).SetBytes(hashHex)
		mask := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
		part1 := new(big.Int).And(val, mask)
		part2 := new(big.Int).Rsh(val, 128)

		b1 := make([]byte, 32)
		part1.FillBytes(b1)
		goMiMC.Write(b1)

		b2 := make([]byte, 32)
		part2.FillBytes(b2)
		goMiMC.Write(b2)
	}
	circuit.BatchRoot = goMiMC.Sum(nil)

	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	if err != nil {
		return nil, nil, err
	}

	publicWitness, err := witness.Public()
	if err != nil {
		return nil, nil, err
	}

	proof, err := groth16.Prove(ccs, pk, witness, backend.WithProverHashToFieldFunction(sha3.NewLegacyKeccak256()))
	if err != nil {
		return nil, nil, err
	}

	return proof, publicWitness, nil
}

func submitProof(client *ethclient.Client, auth *bind.TransactOpts, contractAddr string, batchNumber uint64, chunkIndex int, chunkTrades []ParsedTrade, proof groth16.Proof, publicWitness witness.Witness) (*types.Transaction, error) {
	bn254Proof, ok := proof.(*bn254.Proof)
	if !ok {
		return nil, fmt.Errorf("invalid proof type")
	}

	// Pack proof into uint256[8]
	proof8 := [8]*big.Int{
		bn254Proof.Ar.X.BigInt(new(big.Int)), bn254Proof.Ar.Y.BigInt(new(big.Int)),
		bn254Proof.Bs.X.A1.BigInt(new(big.Int)), bn254Proof.Bs.X.A0.BigInt(new(big.Int)),
		bn254Proof.Bs.Y.A1.BigInt(new(big.Int)), bn254Proof.Bs.Y.A0.BigInt(new(big.Int)),
		bn254Proof.Krs.X.BigInt(new(big.Int)), bn254Proof.Krs.Y.BigInt(new(big.Int)),
	}

	// Extract commitments if they exist
	commitments := [2]*big.Int{big.NewInt(0), big.NewInt(0)}
	if len(bn254Proof.Commitments) > 0 {
		commitments[0] = bn254Proof.Commitments[0].X.BigInt(new(big.Int))
		commitments[1] = bn254Proof.Commitments[0].Y.BigInt(new(big.Int))
	}

	commitmentPok := [2]*big.Int{
		bn254Proof.CommitmentPok.X.BigInt(new(big.Int)),
		bn254Proof.CommitmentPok.Y.BigInt(new(big.Int)),
	}

	// Extract 120 public inputs from publicWitness
	var buf bytes.Buffer
	if _, err := publicWitness.WriteTo(&buf); err != nil {
		return nil, err
	}
	data := buf.Bytes()
	if len(data) < 12 {
		return nil, fmt.Errorf("invalid witness data")
	}

	nbPublic := binary.BigEndian.Uint32(data[0:4])
	if nbPublic != 121 {
		return nil, fmt.Errorf("expected 121 public inputs, got %d", nbPublic)
	}

	var publicInputs [121]*big.Int
	offset := 12
	for i := 0; i < 121; i++ {
		if offset+32 > len(data) {
			return nil, fmt.Errorf("unexpected end of witness data")
		}
		publicInputs[i] = new(big.Int).SetBytes(data[offset : offset+32])
		offset += 32
	}

	var messageHashes [25][32]byte
	var signers [25]common.Address

	for i := 0; i < 25; i++ {
		if i < len(chunkTrades) {
			hashBytes, _ := hex.DecodeString(chunkTrades[i].MessageHash)
			copy(messageHashes[i][:], hashBytes)
			signers[i] = common.HexToAddress(chunkTrades[i].SignerAddress)
		} else {
			hashBytes, _ := hex.DecodeString(chunkTrades[0].MessageHash)
			copy(messageHashes[i][:], hashBytes)
			signers[i] = common.HexToAddress(chunkTrades[0].SignerAddress)
		}
	}

	const abiJSON = `[{"inputs":[{"internalType":"uint256","name":"batchNumber","type":"uint256"},{"internalType":"uint256","name":"chunkIndex","type":"uint256"},{"internalType":"uint256[8]","name":"proof","type":"uint256[8]"},{"internalType":"uint256[2]","name":"commitments","type":"uint256[2]"},{"internalType":"uint256[2]","name":"commitmentPok","type":"uint256[2]"},{"internalType":"uint256[301]","name":"publicInputs","type":"uint256[301]"},{"internalType":"bytes32[25]","name":"messageHashes","type":"bytes32[25]"},{"internalType":"address[25]","name":"signers","type":"address[25]"}],"name":"registerTrades","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	contract := bind.NewBoundContract(common.HexToAddress(contractAddr), parsedABI, client, client, client)

	// Dump calldata
	calldata, _ := parsedABI.Pack("registerTrades", new(big.Int).SetUint64(batchNumber), new(big.Int).SetInt64(int64(chunkIndex)), proof8, commitments, commitmentPok, publicInputs, messageHashes, signers)
	os.WriteFile("calldata.hex", []byte(hex.EncodeToString(calldata)), 0644)

	// Set a high manual gas limit so go-ethereum skips eth_estimateGas
	// This will let the transaction hit the chain and revert, allowing us to debug it via txhash.
	auth.GasLimit = 15000000

	tx, err := contract.Transact(auth, "registerTrades", new(big.Int).SetUint64(batchNumber), new(big.Int).SetInt64(int64(chunkIndex)), proof8, commitments, commitmentPok, publicInputs, messageHashes, signers)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	return tx, nil
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

// ErrorType represents the type of error encountered
type ErrorType int

const (
	ErrorTypeNetwork ErrorType = iota
	ErrorTypeVerification
	ErrorTypeDuplicate // Batch already exists on L1
	ErrorTypeUnknown
)

// classifyError determines the type of error from submission/transaction
func classifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	errMsg := err.Error()

	// Duplicate batch error (already submitted to L1)
	if strings.Contains(errMsg, "batch hash already exists") ||
		strings.Contains(errMsg, "batch already exists") {
		return ErrorTypeDuplicate
	}

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
	failureDir := os.ExpandEnv("$HOME/.nowa-zk/prover/failures")
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
	log.Println("1. Review failure data in: ~/.nowa-zk/prover/failures/")
	log.Printf("   - Batch data: batch_%d.json\n", batch.Number)
	log.Printf("   - Proof: batch_%d_proof.bin\n", batch.Number)
	log.Printf("   - Witness: batch_%d_witness.bin\n", batch.Number)
	log.Printf("   - Error log: batch_%d_error.log\n", batch.Number)
	log.Println("2. Verify circuit constraints match contract verifier")
	log.Println("3. Check for non-deterministic proof generation bugs")
	log.Println("4. Ensure state synchronization is correct")
	log.Println("5. After fixing, restart with --clear-halt flag:")
	log.Println("   make run-prover CLEAR_HALT=true")
	log.Println("   OR: ./build/prover-bin start --keys-dir ~/.nowa-zk/keys --clear-halt")
	log.Println()
	log.Println("For support, visit: https://github.com/nowafinance/nowa-zk/issues")
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
	chunkIndex int,
	trades []ParsedTrade,
	ccs constraint.ConstraintSystem,
	pk groth16.ProvingKey,
	vk groth16.VerifyingKey,
	store *storage.ProverStore,
	enableParanoid bool,
	maxRebuilds int,
	testMode bool,
) (*types.Transaction, *types.Receipt, error) {

	// Level 1: Try with initial proof (3 quick retries for network issues)
	proof, publicWitness, err := generateProof(trades, ccs, pk)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	log.Println("   ✅ Proof generated")

	// Verify locally
	log.Println("   🔍 Verifying proof locally...")
	if err := verifyLocal(proof, vk, publicWitness); err != nil {
		return nil, nil, fmt.Errorf("local verification failed: %w", err)
	}
	log.Println("   ✅ Proof verified locally")

	// Attempt submission with retries
	tx, receipt, err := attemptSubmission(client, auth, contractAddr, batch.Number, chunkIndex, trades, proof, publicWitness, 3, testMode)
	if err == nil {
		return tx, receipt, nil
	}

	errType := classifyError(err)
	log.Printf("   ⚠️  Submission failed: %v (Type: %v)\n", err, errType)

	if errType == ErrorTypeDuplicate {
		log.Println("   ℹ️  Batch chunk already exists on L1 - skipping to next")
		return tx, receipt, nil
	}
	if errType == ErrorTypeNetwork {
		return nil, nil, fmt.Errorf("network error after retries: %w", err)
	}
	if !enableParanoid || errType != ErrorTypeVerification {
		return nil, nil, err
	}

	log.Println()
	log.Println("   🔄 PARANOID MODE: Rebuilding proof due to verification failure...")

	for rebuild := 1; rebuild <= maxRebuilds; rebuild++ {
		log.Printf("   🔧 Rebuild attempt %d/%d\n", rebuild, maxRebuilds)

		proof, publicWitness, err = generateProof(trades, ccs, pk)
		if err != nil {
			log.Printf("   ❌ Proof regeneration failed: %v\n", err)
			continue
		}
		log.Println("   ✅ Proof regenerated")

		if err := verifyLocal(proof, vk, publicWitness); err != nil {
			log.Printf("   ❌ Local verification failed: %v\n", err)
			continue
		}
		log.Println("   ✅ Proof verified locally")

		tx, receipt, err = attemptSubmission(client, auth, contractAddr, batch.Number, chunkIndex, trades, proof, publicWitness, 3, testMode)
		if err == nil {
			log.Println("   ✅ Proof accepted after rebuild!")
			return tx, receipt, nil
		}
		log.Printf("   ❌ Rebuilt proof also failed: %v\n", err)
	}

	errMsg := fmt.Sprintf("Verification failed after %d rebuild attempts: %v", maxRebuilds, err)
	if saveErr := saveFailureData(batch, proof, publicWitness, errMsg); saveErr != nil {
		log.Printf("⚠️  Failed to save failure data: %v\n", saveErr)
	}
	var proofBuf bytes.Buffer
	proof.WriteTo(&proofBuf)
	if saveErr := store.SaveVerificationFailure(batch.Number, errMsg, proofBuf.Bytes()); saveErr != nil {
		log.Printf("⚠️  Failed to save failure to storage: %v\n", saveErr)
	}
	haltProver(store, batch, errMsg)

	return nil, nil, fmt.Errorf("HALTED: %s", errMsg)
}

// attemptSubmission tries to submit and verify a proof with retries
func attemptSubmission(
	client *ethclient.Client,
	auth *bind.TransactOpts,
	contractAddr string,
	batchNumber uint64,
	chunkIndex int,
	chunkTrades []ParsedTrade,
	proof groth16.Proof,
	publicWitness witness.Witness,
	maxAttempts int,
	testMode bool,
) (*types.Transaction, *types.Receipt, error) {

	var tx *types.Transaction
	var err error

	proofToSubmit := proof
	if testMode {
		log.Println("   🧪 TEST MODE: Corrupting proof to simulate verification failure...")
		bn254Proof, ok := proof.(*bn254.Proof)
		if ok {
			corruptedProof := *bn254Proof
			corruptedProof.Ar.X.Add(&corruptedProof.Ar.X, &corruptedProof.Ar.X)
			proofToSubmit = &corruptedProof
		}
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		tx, err = submitProof(client, auth, contractAddr, batchNumber, chunkIndex, chunkTrades, proofToSubmit, publicWitness)
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
