package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
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
	"github.com/nowafinance/nowa-zk/prover/circuits"
	"github.com/nowafinance/nowa-zk/prover/internal/storage"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
)

type StateUpdate struct {
	Index    uint64   `json:"index"`
	Balance  string   `json:"balance"`
	Nonce    uint64   `json:"nonce"`
	Path     [28]string `json:"path"`
	PathBits []bool   `json:"pathBits"`
}

type StateTransition struct {
	OpType int `json:"op_type"`

	Amount      string `json:"amount"`
	QuoteAmount string `json:"quote_amount"`

	MakerPubKeyX string      `json:"maker_pub_key_x"`
	MakerPubKeyY string      `json:"maker_pub_key_y"`
	MakerSig     string      `json:"maker_sig"`
	MakerBase    StateUpdate `json:"maker_base"`
	MakerQuote   StateUpdate `json:"maker_quote"`

	TakerPubKeyX string      `json:"taker_pub_key_x"`
	TakerPubKeyY string      `json:"taker_pub_key_y"`
	TakerSig     string      `json:"taker_sig"`
	TakerBase    StateUpdate `json:"taker_base"`
	TakerQuote   StateUpdate `json:"taker_quote"`
}

type ZKBatch struct {
	BatchID        uint64            `json:"batch_id"`
	OldRoot        string            `json:"old_root"`
	NewRoot        string            `json:"new_root"`
	WithdrawalHash string            `json:"withdrawal_hash"`
	DepositHash    string            `json:"deposit_hash"`
	Transitions    []StateTransition `json:"transitions"`
}

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
	startCmd.Flags().StringVarP(&keysDir, "keys-dir", "k", "./keys", "")
	startCmd.Flags().StringVarP(&dataDir, "data-dir", "d", "", "")
	startCmd.Flags().StringVarP(&indexerURL, "indexer-url", "s", "http://localhost:8080", "")
	startCmd.Flags().StringVarP(&rpcURL, "rpc-url", "r", "", "Ethereum RPC URL")
	startCmd.Flags().StringVarP(&contractAddr, "contract", "c", "", "")
	startCmd.Flags().StringVarP(&privateKeyHex, "private-key", "p", "", "")
	startCmd.Flags().IntVarP(&pollInterval, "poll-interval", "i", 10, "")
	startCmd.Flags().StringVar(&configPath, "config", "", "Path to YAML config file")
	startCmd.Flags().BoolVar(&enableParanoidMode, "paranoid-mode", true, "Enable proof rebuild on verification failure")
	startCmd.Flags().IntVar(&maxRebuildAttempts, "max-rebuilds", 1, "Maximum proof rebuild attempts")
	startCmd.Flags().BoolVar(&clearHalt, "clear-halt", false, "Clear halt state and resume processing")
	startCmd.Flags().BoolVar(&testFailure, "test-failure", false, "[TESTING] Intentionally corrupt proof to test error handling")
}

func start(cmd *cobra.Command, args []string) {
	_ = godotenv.Load()

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

	var loadedDeployments map[string]string
	localDeployPath := ".nowa-zk/deployments.json"
	if data, err := os.ReadFile(localDeployPath); err == nil {
		if err := json.Unmarshal(data, &loadedDeployments); err == nil {
			if contractAddr == "" {
				if addr, ok := loadedDeployments["NowaRollup"]; ok {
					contractAddr = addr
				}
			}
		}
	}

	if contractAddr == "" {
		log.Fatal("❌ Contract address required.")
	}
	if privateKeyHex == "" {
		log.Fatal("❌ Private key required.")
	}
	log.Println("========================================")
	log.Println("  ZK Rollup Prover Service")
	log.Println("========================================")

	ccs, pk, vk, err := loadCircuitAndKeys()
	if err != nil {
		log.Fatalf("❌ Failed to load circuit/keys: %v", err)
	}

	client, auth, err := connectEthereum(rpcURL, privateKeyHex)
	if err != nil {
		log.Fatalf("❌ Failed to connect to Ethereum: %v", err)
	}

	var storePath string
	if dataDir != "" {
		storePath = dataDir
	} else {
		homeDir, _ := os.UserHomeDir()
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

	if clearHalt {
		_ = store.ClearHaltState()
	}

	lastProcessedBatch, err := store.GetLastProcessedBatch()
	if err != nil {
		lastProcessedBatch = 0
	}

	log.Println("🚀 Starting prover loop...")
	for {
		latestBatch, err := fetchLatestBatch(indexerURL)
		if err != nil || latestBatch == nil {
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		nextBatchNum := lastProcessedBatch + 1
		if nextBatchNum > latestBatch.BatchID {
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		log.Printf("📦 Processing batch #%d\n", nextBatchNum)
		batch, err := fetchLatestBatch(indexerURL) // We just fetch the latest for now in this demo since it's 1-by-1
		if err != nil {
			log.Printf("⚠️  Failed to fetch batch: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		if len(batch.Transitions) == 0 {
			log.Printf("   ℹ️  No operations in batch #%d, skipping...\n", batch.BatchID)
			_ = store.SaveLastProcessedBatch(batch.BatchID)
			lastProcessedBatch = batch.BatchID
			continue
		}

		log.Println("   🔐 Generating proof...")
		proof, publicWitness, err := generateProof(batch, ccs, pk)
		if err != nil {
			log.Printf("   ❌ Failed to generate proof: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		
		log.Println("   🕵️‍♂️ Verifying proof locally...")
		err = verifyLocal(proof, vk, publicWitness)
		if err != nil {
			log.Printf("   ❌ Local Verification Failed: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		log.Println("   📤 Submitting proof to L1...")
		_, _, err = submitProof(client, auth, contractAddr, batch, proof, publicWitness)
		if err != nil {
			log.Printf("   ❌ Failed to submit to L1: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		log.Printf("   ✅ Batch #%d successfully proven and submitted!\n", batch.BatchID)
		_ = store.SaveLastProcessedBatch(batch.BatchID)
		lastProcessedBatch = batch.BatchID
	}
}

func loadCircuitAndKeys() (constraint.ConstraintSystem, groth16.ProvingKey, groth16.VerifyingKey, error) {
	ccsFile, err := os.Open(keysDir + "/state.ccs")
	if err != nil { return nil, nil, nil, err }
	defer ccsFile.Close()
	ccs := groth16.NewCS(ecc.BN254)
	if _, err := ccs.ReadFrom(ccsFile); err != nil { return nil, nil, nil, err }

	pkFile, err := os.Open(keysDir + "/state.pk")
	if err != nil { return nil, nil, nil, err }
	defer pkFile.Close()
	pk := groth16.NewProvingKey(ecc.BN254)
	if _, err := pk.ReadFrom(pkFile); err != nil { return nil, nil, nil, err }

	vkFile, err := os.Open(keysDir + "/state.vk")
	if err != nil { return nil, nil, nil, err }
	defer vkFile.Close()
	vk := groth16.NewVerifyingKey(ecc.BN254)
	if _, err := vk.ReadFrom(vkFile); err != nil { return nil, nil, nil, err }

	return ccs, pk, vk, nil
}

func connectEthereum(rpcURL, privateKeyHex string) (*ethclient.Client, *bind.TransactOpts, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil { return nil, nil, err }

	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil { return nil, nil, err }

	chainID, err := client.NetworkID(context.Background())
	if err != nil { return nil, nil, err }

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil { return nil, nil, err }

	return client, auth, nil
}

func fetchLatestBatch(indexerURL string) (*ZKBatch, error) {
	resp, err := http.Get(indexerURL + "/batch/latest")
	if err != nil { return nil, err }
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var batch ZKBatch
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return nil, err
	}
	return &batch, nil
}

func assignPath(target *[28]frontend.Variable, targetBits *[28]frontend.Variable, pathStr [28]string, bits []bool) {
	for j := 0; j < 28; j++ {
		b, _ := new(big.Int).SetString(pathStr[j], 10)
		target[j] = b
		if bits[j] {
			targetBits[j] = 1
		} else {
			targetBits[j] = 0
		}
	}
}

func generateProof(batch *ZKBatch, ccs constraint.ConstraintSystem, pk groth16.ProvingKey) (groth16.Proof, witness.Witness, error) {
	var circuit circuits.StateTransitionCircuit

	oldRootHex := strings.TrimPrefix(batch.OldRoot, "0x")
	oldRoot, _ := new(big.Int).SetString(oldRootHex, 16)
	if oldRoot == nil { oldRoot, _ = new(big.Int).SetString(batch.OldRoot, 10) }

	newRootHex := strings.TrimPrefix(batch.NewRoot, "0x")
	newRoot, _ := new(big.Int).SetString(newRootHex, 16)
	if newRoot == nil { newRoot, _ = new(big.Int).SetString(batch.NewRoot, 10) }

	withdrawalHash, _ := new(big.Int).SetString(batch.WithdrawalHash, 10)
	depositHash, _ := new(big.Int).SetString(batch.DepositHash, 10)

	circuit.OldRoot = oldRoot
	circuit.NewRoot = newRoot
	circuit.WithdrawalHash = withdrawalHash
	circuit.DepositHash = depositHash

	for i := 0; i < circuits.BatchSize; i++ {
		var op StateTransition
		if i < len(batch.Transitions) {
			op = batch.Transitions[i]
		} else {
			op = batch.Transitions[0] // Pad with first operation just to not be empty (will fail if we don't have valid padding, but for demo we just copy)
		}

		circuit.Ops[i].OpType = op.OpType
		
		circuit.Ops[i].Amount, _ = new(big.Int).SetString(op.Amount, 10)
		circuit.Ops[i].QuoteAmount, _ = new(big.Int).SetString(op.QuoteAmount, 10)

		// Maker
		circuit.Ops[i].MakerPubKey.A.X, _ = new(big.Int).SetString(op.MakerPubKeyX, 10)
		circuit.Ops[i].MakerPubKey.A.Y, _ = new(big.Int).SetString(op.MakerPubKeyY, 10)
		
		circuit.Ops[i].MakerBase.Index = op.MakerBase.Index
		circuit.Ops[i].MakerBase.Balance, _ = new(big.Int).SetString(op.MakerBase.Balance, 10)
		circuit.Ops[i].MakerBase.Nonce = op.MakerBase.Nonce
		assignPath(&circuit.Ops[i].MakerBase.Path, &circuit.Ops[i].MakerBase.PathBits, op.MakerBase.Path, op.MakerBase.PathBits)

		circuit.Ops[i].MakerQuote.Index = op.MakerQuote.Index
		circuit.Ops[i].MakerQuote.Balance, _ = new(big.Int).SetString(op.MakerQuote.Balance, 10)
		circuit.Ops[i].MakerQuote.Nonce = op.MakerQuote.Nonce
		assignPath(&circuit.Ops[i].MakerQuote.Path, &circuit.Ops[i].MakerQuote.PathBits, op.MakerQuote.Path, op.MakerQuote.PathBits)

		// Taker
		circuit.Ops[i].TakerPubKey.A.X, _ = new(big.Int).SetString(op.TakerPubKeyX, 10)
		circuit.Ops[i].TakerPubKey.A.Y, _ = new(big.Int).SetString(op.TakerPubKeyY, 10)
		
		circuit.Ops[i].TakerBase.Index = op.TakerBase.Index
		circuit.Ops[i].TakerBase.Balance, _ = new(big.Int).SetString(op.TakerBase.Balance, 10)
		circuit.Ops[i].TakerBase.Nonce = op.TakerBase.Nonce
		assignPath(&circuit.Ops[i].TakerBase.Path, &circuit.Ops[i].TakerBase.PathBits, op.TakerBase.Path, op.TakerBase.PathBits)

		circuit.Ops[i].TakerQuote.Index = op.TakerQuote.Index
		circuit.Ops[i].TakerQuote.Balance, _ = new(big.Int).SetString(op.TakerQuote.Balance, 10)
		circuit.Ops[i].TakerQuote.Nonce = op.TakerQuote.Nonce
		assignPath(&circuit.Ops[i].TakerQuote.Path, &circuit.Ops[i].TakerQuote.PathBits, op.TakerQuote.Path, op.TakerQuote.PathBits)
	}

	witness, err := frontend.NewWitness(&circuit, ecc.BN254.ScalarField())
	if err != nil { return nil, nil, err }

	publicWitness, err := witness.Public()
	if err != nil { return nil, nil, err }

	proof, err := groth16.Prove(ccs, pk, witness, backend.WithProverHashToFieldFunction(sha3.NewLegacyKeccak256()))
	return proof, publicWitness, err
}

func verifyLocal(proof groth16.Proof, vk groth16.VerifyingKey, publicWitness witness.Witness) error {
	return groth16.Verify(proof, vk, publicWitness, backend.WithVerifierHashToFieldFunction(sha3.NewLegacyKeccak256()))
}

func submitProof(client *ethclient.Client, auth *bind.TransactOpts, contractAddr string, batch *ZKBatch, proof groth16.Proof, publicWitness witness.Witness) (*types.Transaction, *types.Receipt, error) {
	bn254Proof, _ := proof.(*bn254.Proof)

	proof8 := [8]*big.Int{
		bn254Proof.Ar.X.BigInt(new(big.Int)), bn254Proof.Ar.Y.BigInt(new(big.Int)),
		bn254Proof.Bs.X.A1.BigInt(new(big.Int)), bn254Proof.Bs.X.A0.BigInt(new(big.Int)),
		bn254Proof.Bs.Y.A1.BigInt(new(big.Int)), bn254Proof.Bs.Y.A0.BigInt(new(big.Int)),
		bn254Proof.Krs.X.BigInt(new(big.Int)), bn254Proof.Krs.Y.BigInt(new(big.Int)),
	}

	const abiJSON = `[{"inputs":[{"internalType":"uint256[8]","name":"proof","type":"uint256[8]"},{"internalType":"uint256[4]","name":"publicInputs","type":"uint256[4]"}],"name":"submitBatch","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
	parsedABI, _ := abi.JSON(strings.NewReader(abiJSON))
	contract := bind.NewBoundContract(common.HexToAddress(contractAddr), parsedABI, client, client, client)

	auth.GasLimit = 15000000

	oldRootHex := strings.TrimPrefix(batch.OldRoot, "0x")
	oldRootBytes, _ := hex.DecodeString(fmt.Sprintf("%064s", oldRootHex))
	var oldRoot [32]byte
	copy(oldRoot[:], oldRootBytes)
	
	newRootHex := strings.TrimPrefix(batch.NewRoot, "0x")
	newRootBytes, _ := hex.DecodeString(fmt.Sprintf("%064s", newRootHex))
	var newRoot [32]byte
	copy(newRoot[:], newRootBytes)
	
	publicInputs := [4]*big.Int{
		new(big.Int).SetBytes(oldRoot[:]),
		new(big.Int).SetBytes(newRoot[:]),
		new(big.Int).SetInt64(0), // WithdrawalHash (not implemented yet)
		new(big.Int).SetInt64(0), // DepositHash (not implemented yet)
	}

	tx, err := contract.Transact(auth, "submitBatch", proof8, publicInputs)
	if err != nil { return nil, nil, err }

	receipt, err := bind.WaitMined(context.Background(), client, tx)
	return tx, receipt, err
}
