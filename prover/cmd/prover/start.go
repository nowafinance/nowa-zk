package main

import (
	"context"
	"crypto/ecdsa"
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
	"github.com/consensys/gnark-crypto/ecc/twistededwards"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	bn254 "github.com/consensys/gnark/backend/groth16/bn254"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	gnarkeddsa "github.com/consensys/gnark/std/signature/eddsa"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/joho/godotenv"
	"github.com/nowafinance/nowa-zk/prover/circuits"
	"github.com/nowafinance/nowa-zk/prover/internal/da"
	"github.com/nowafinance/nowa-zk/prover/internal/storage"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/sha3"
)

type StateUpdate struct {
	Index    uint64     `json:"index"`
	Balance  string     `json:"balance"`
	Nonce    uint64     `json:"nonce"`
	Path     [28]string `json:"path"`
	PathBits [28]uint64 `json:"path_bits"` // was []bool — fixed to match sequencer JSON
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
	if contractAddr == "" {
		if env := os.Getenv("ROLLUP_CONTRACT_ADDRESS"); env != "" {
			contractAddr = env
		} else if env := os.Getenv("CONTRACT_ADDRESS"); env != "" {
			contractAddr = env
		}
	}
	if contractAddr == "" {
		home, _ := os.UserHomeDir()
		for _, path := range []string{
			".nowa-zk/deployments.json",
			filepath.Join(home, ".nowa-zk", "deployments.json"),
			"contracts/deployments/deployments.json",
		} {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if err := json.Unmarshal(data, &loadedDeployments); err != nil {
				continue
			}
			if addr, ok := loadedDeployments["NowaRollup"]; ok && addr != "" && addr != "null" {
				contractAddr = addr
				log.Printf("📄 Loaded NowaRollup from %s", path)
				break
			}
		}
	}

	if contractAddr == "" {
		log.Fatal("❌ Contract address required. Pass --contract, set ROLLUP_CONTRACT_ADDRESS, or deploy so ~/.nowa-zk/deployments.json exists.")
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

	client, auth, privKey, err := connectEthereum(rpcURL, privateKeyHex)
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
			// No new batch yet
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}

		log.Printf("📦 Processing batch #%d\n", nextBatchNum)
		batch, err := fetchBatchByID(indexerURL, nextBatchNum)
		if err != nil || batch == nil {
			log.Printf("⚠️  Failed to fetch batch #%d: %v\n", nextBatchNum, err)
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

		log.Println("   📤 Submitting proof + EIP-4844 DA blob to L1...")
		tx, receipt, err := submitProofWithBlob(client, auth, privKey, contractAddr, batch, proof)
		if err != nil {
			log.Printf("   ❌ Failed to submit to L1: %v\n", err)
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		if receipt != nil && receipt.Status == 0 {
			log.Printf("   ❌ L1 tx reverted: %s\n", tx.Hash().Hex())
			time.Sleep(time.Duration(pollInterval) * time.Second)
			continue
		}
		log.Printf("   📎 L1 tx %s (blob DA included)\n", tx.Hash().Hex())

		log.Printf("   ✅ Batch #%d successfully proven and submitted!\n", batch.BatchID)
		_ = store.SaveLastProcessedBatch(batch.BatchID)
		lastProcessedBatch = batch.BatchID
	}
}

func loadCircuitAndKeys() (constraint.ConstraintSystem, groth16.ProvingKey, groth16.VerifyingKey, error) {
	// Load compiled R1CS constraint system.
	// The setup command writes these as rollup.r1cs / rollup.pk / rollup.vk.
	ccsFile, err := os.Open(keysDir + "/rollup.r1cs")
	if err != nil {
		// Fallback to legacy name produced by older setup runs.
		ccsFile, err = os.Open(keysDir + "/state.ccs")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("cannot open circuit file (tried rollup.r1cs and state.ccs): %w", err)
		}
	}
	defer ccsFile.Close()
	ccs := groth16.NewCS(ecc.BN254)
	if _, err := ccs.ReadFrom(ccsFile); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read constraint system: %w", err)
	}

	pkFile, err := os.Open(keysDir + "/rollup.pk")
	if err != nil {
		pkFile, err = os.Open(keysDir + "/state.pk")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("cannot open proving key (tried rollup.pk and state.pk): %w", err)
		}
	}
	defer pkFile.Close()
	pk := groth16.NewProvingKey(ecc.BN254)
	if _, err := pk.ReadFrom(pkFile); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read proving key: %w", err)
	}

	vkFile, err := os.Open(keysDir + "/rollup.vk")
	if err != nil {
		vkFile, err = os.Open(keysDir + "/state.vk")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("cannot open verifying key (tried rollup.vk and state.vk): %w", err)
		}
	}
	defer vkFile.Close()
	vk := groth16.NewVerifyingKey(ecc.BN254)
	if _, err := vk.ReadFrom(vkFile); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to read verifying key: %w", err)
	}

	return ccs, pk, vk, nil
}

func connectEthereum(rpcURL, privateKeyHex string) (*ethclient.Client, *bind.TransactOpts, *ecdsa.PrivateKey, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, nil, nil, err
	}

	if len(privateKeyHex) > 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, nil, nil, err
	}

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return nil, nil, nil, err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, nil, nil, err
	}

	return client, auth, privateKey, nil
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

// fetchBatchByID fetches a specific batch from the sequencer by its ID.
func fetchBatchByID(indexerURL string, id uint64) (*ZKBatch, error) {
	resp, err := http.Get(fmt.Sprintf("%s/batch/%d", indexerURL, id))
	if err != nil { return nil, err }
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sequencer API error for batch %d: %s", id, string(body))
	}

	var batch ZKBatch
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return nil, err
	}
	return &batch, nil
}

// assignPath copies a Merkle path and its direction bits into circuit variables.
func assignPath(target *[28]frontend.Variable, targetBits *[28]frontend.Variable, pathStr [28]string, bits [28]uint64) {
	for j := 0; j < 28; j++ {
		b, _ := new(big.Int).SetString(pathStr[j], 10)
		target[j] = b
		targetBits[j] = bits[j] // uint64 is directly assignable to frontend.Variable
	}
}

// assignEdDSASig decodes a 64-byte compressed EdDSA signature (R||S) into circuit vars.
// Critical: R is a compressed twisted-Edwards point — must decompress Y, not set RY=0.
func assignEdDSASig(dst *gnarkeddsa.Signature, sigHex string) error {
	clean := strings.TrimPrefix(sigHex, "0x")
	sigBytes, err := hex.DecodeString(clean)
	if err != nil {
		return err
	}
	if len(sigBytes) != 64 && len(sigBytes) != 96 {
		return fmt.Errorf("unexpected signature length %d", len(sigBytes))
	}
	dst.Assign(twistededwards.BN254, sigBytes)
	return nil
}

func generateProof(batch *ZKBatch, ccs constraint.ConstraintSystem, pk groth16.ProvingKey) (groth16.Proof, witness.Witness, error) {
	var circuit circuits.StateTransitionCircuit

	oldRoot, _ := new(big.Int).SetString(batch.OldRoot, 10)
	if oldRoot == nil {
		oldRootHex := strings.TrimPrefix(batch.OldRoot, "0x")
		oldRoot, _ = new(big.Int).SetString(oldRootHex, 16)
	}
	newRoot, _ := new(big.Int).SetString(batch.NewRoot, 10)
	if newRoot == nil {
		newRootHex := strings.TrimPrefix(batch.NewRoot, "0x")
		newRoot, _ = new(big.Int).SetString(newRootHex, 16)
	}

	withdrawalHash, _ := new(big.Int).SetString(batch.WithdrawalHash, 10)
	if withdrawalHash == nil {
		withdrawalHash = big.NewInt(0)
	}
	depositHash, _ := new(big.Int).SetString(batch.DepositHash, 10)
	if depositHash == nil {
		depositHash = big.NewInt(0)
	}

	circuit.OldRoot = oldRoot
	circuit.NewRoot = newRoot
	circuit.WithdrawalHash = withdrawalHash
	circuit.DepositHash = depositHash

	for i := 0; i < circuits.BatchSize; i++ {
		var op StateTransition
		if i < len(batch.Transitions) {
			op = batch.Transitions[i]
		} else {
			op = batch.Transitions[len(batch.Transitions)-1]
		}

		circuit.Ops[i].OpType = op.OpType
		circuit.Ops[i].Amount, _ = new(big.Int).SetString(op.Amount, 10)
		circuit.Ops[i].QuoteAmount, _ = new(big.Int).SetString(op.QuoteAmount, 10)

		circuit.Ops[i].MakerPubKey.A.X, _ = new(big.Int).SetString(op.MakerPubKeyX, 10)
		circuit.Ops[i].MakerPubKey.A.Y, _ = new(big.Int).SetString(op.MakerPubKeyY, 10)
		if err := assignEdDSASig(&circuit.Ops[i].MakerSig, op.MakerSig); err != nil {
			circuit.Ops[i].MakerSig.R.X = 0
			circuit.Ops[i].MakerSig.R.Y = 0
			circuit.Ops[i].MakerSig.S = 0
		}

		circuit.Ops[i].MakerBase.Index = op.MakerBase.Index
		circuit.Ops[i].MakerBase.Balance, _ = new(big.Int).SetString(op.MakerBase.Balance, 10)
		circuit.Ops[i].MakerBase.Nonce = op.MakerBase.Nonce
		assignPath(&circuit.Ops[i].MakerBase.Path, &circuit.Ops[i].MakerBase.PathBits, op.MakerBase.Path, op.MakerBase.PathBits)

		circuit.Ops[i].MakerQuote.Index = op.MakerQuote.Index
		circuit.Ops[i].MakerQuote.Balance, _ = new(big.Int).SetString(op.MakerQuote.Balance, 10)
		circuit.Ops[i].MakerQuote.Nonce = op.MakerQuote.Nonce
		assignPath(&circuit.Ops[i].MakerQuote.Path, &circuit.Ops[i].MakerQuote.PathBits, op.MakerQuote.Path, op.MakerQuote.PathBits)

		circuit.Ops[i].TakerPubKey.A.X, _ = new(big.Int).SetString(op.TakerPubKeyX, 10)
		circuit.Ops[i].TakerPubKey.A.Y, _ = new(big.Int).SetString(op.TakerPubKeyY, 10)
		if err := assignEdDSASig(&circuit.Ops[i].TakerSig, op.TakerSig); err != nil {
			circuit.Ops[i].TakerSig.R.X = 0
			circuit.Ops[i].TakerSig.R.Y = 0
			circuit.Ops[i].TakerSig.S = 0
		}

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
	if err != nil {
		return nil, nil, err
	}

	publicWitness, err := witness.Public()
	if err != nil {
		return nil, nil, err
	}

	proof, err := groth16.Prove(ccs, pk, witness, backend.WithProverHashToFieldFunction(sha3.NewLegacyKeccak256()))
	return proof, publicWitness, err
}

func verifyLocal(proof groth16.Proof, vk groth16.VerifyingKey, publicWitness witness.Witness) error {
	return groth16.Verify(proof, vk, publicWitness, backend.WithVerifierHashToFieldFunction(sha3.NewLegacyKeccak256()))
}

func rootToBytes32(root string) ([32]byte, error) {
	var out [32]byte
	var b *big.Int
	var ok bool

	trimmed := strings.TrimSpace(root)
	if strings.HasPrefix(trimmed, "0x") || strings.HasPrefix(trimmed, "0X") {
		b, ok = new(big.Int).SetString(trimmed[2:], 16)
	} else {
		// Sequencer emits decimal Fr elements. Digits-only strings are also valid hex,
		// so decimal MUST be tried first or FillBytes panics on oversized values.
		b, ok = new(big.Int).SetString(trimmed, 10)
		if !ok {
			b, ok = new(big.Int).SetString(trimmed, 16)
		}
	}
	if !ok || b == nil {
		return out, fmt.Errorf("invalid root %q", root)
	}
	if b.Sign() < 0 || b.BitLen() > 256 {
		return out, fmt.Errorf("root out of bytes32 range (bitlen=%d): %q", b.BitLen(), root)
	}
	b.FillBytes(out[:])
	return out, nil
}

func hashToBytes32(v string) [32]byte {
	var out [32]byte
	if v == "" || v == "0" {
		return out
	}
	b, err := rootToBytes32(v)
	if err != nil {
		return out
	}
	return b
}

// submitProofWithBlob posts the batch DA payload in an EIP-4844 blob and calls submitBatch.
func submitProofWithBlob(
	client *ethclient.Client,
	auth *bind.TransactOpts,
	privKey *ecdsa.PrivateKey,
	contractAddr string,
	batch *ZKBatch,
	proof groth16.Proof,
) (*types.Transaction, *types.Receipt, error) {
	bn254Proof, ok := proof.(*bn254.Proof)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected proof type %T", proof)
	}

	proof8 := [8]*big.Int{
		bn254Proof.Ar.X.BigInt(new(big.Int)), bn254Proof.Ar.Y.BigInt(new(big.Int)),
		bn254Proof.Bs.X.A1.BigInt(new(big.Int)), bn254Proof.Bs.X.A0.BigInt(new(big.Int)),
		bn254Proof.Bs.Y.A1.BigInt(new(big.Int)), bn254Proof.Bs.Y.A0.BigInt(new(big.Int)),
		bn254Proof.Krs.X.BigInt(new(big.Int)), bn254Proof.Krs.Y.BigInt(new(big.Int)),
	}

	wHash := batch.WithdrawalHash
	dHash := batch.DepositHash
	if wHash == "" {
		wHash = "0"
	}
	if dHash == "" {
		dHash = "0"
	}

	payload, dataHash, err := da.EncodeBatchPayload(batch.BatchID, batch.OldRoot, batch.NewRoot, wHash, dHash, batch.Transitions)
	if err != nil {
		return nil, nil, err
	}
	sidecar, blobHash, err := da.BuildBlobSidecar(payload)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("   📦 DA payload %d bytes, dataHash=%s blobHash=%s\n", len(payload), dataHash.Hex(), blobHash.Hex())

	oldRoot, err := rootToBytes32(batch.OldRoot)
	if err != nil {
		return nil, nil, err
	}
	newRoot, err := rootToBytes32(batch.NewRoot)
	if err != nil {
		return nil, nil, err
	}
	withdrawalHash := hashToBytes32(wHash)
	depositHash := hashToBytes32(dHash)

	const abiJSON = `[{"inputs":[{"internalType":"uint256[8]","name":"proof","type":"uint256[8]"},{"internalType":"bytes32","name":"_oldRoot","type":"bytes32"},{"internalType":"bytes32","name":"_newRoot","type":"bytes32"},{"internalType":"bytes32","name":"_withdrawalHash","type":"bytes32"},{"internalType":"bytes32","name":"_depositHash","type":"bytes32"},{"internalType":"bytes32","name":"_dataHash","type":"bytes32"}],"name":"submitBatch","outputs":[],"stateMutability":"nonpayable","type":"function"}]`
	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return nil, nil, err
	}
	calldata, err := parsedABI.Pack("submitBatch", proof8, oldRoot, newRoot, withdrawalHash, depositHash, dataHash)
	if err != nil {
		return nil, nil, fmt.Errorf("abi pack: %w", err)
	}

	ctx := context.Background()
	from := auth.From
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		return nil, nil, err
	}
	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, nil, err
	}
	tip, err := client.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, nil, err
	}
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	baseFee := header.BaseFee
	if baseFee == nil {
		baseFee = big.NewInt(1)
	}
	// maxFee = 2*baseFee + tip
	gasFeeCap := new(big.Int).Mul(baseFee, big.NewInt(2))
	gasFeeCap.Add(gasFeeCap, tip)

	blobFee := big.NewInt(params.BlobTxMinBlobGasprice)
	if header.ExcessBlobGas != nil {
		cfg := params.MainnetChainConfig
		if chainID.Cmp(big.NewInt(11155111)) == 0 {
			cfg = params.SepoliaChainConfig
		}
		blobFee = eip4844.CalcBlobFee(cfg, header)
	}
	blobFeeCap := new(big.Int).Mul(blobFee, big.NewInt(2))
	if blobFeeCap.Cmp(big.NewInt(1)) < 0 {
		blobFeeCap = big.NewInt(1)
	}

	to := common.HexToAddress(contractAddr)
	gasLimit := uint64(1_500_000)
	if auth.GasLimit != 0 {
		gasLimit = auth.GasLimit
	}

	blobTx := &types.BlobTx{
		ChainID:    uint256.MustFromBig(chainID),
		Nonce:      nonce,
		GasTipCap:  uint256.MustFromBig(tip),
		GasFeeCap:  uint256.MustFromBig(gasFeeCap),
		Gas:        gasLimit,
		To:         to,
		Value:      uint256.NewInt(0),
		Data:       calldata,
		BlobFeeCap: uint256.MustFromBig(blobFeeCap),
		BlobHashes: sidecar.BlobHashes(),
		Sidecar:    sidecar,
	}

	signer := types.NewCancunSigner(chainID)
	signed, err := types.SignNewTx(privKey, signer, blobTx)
	if err != nil {
		return nil, nil, fmt.Errorf("sign blob tx: %w", err)
	}

	if err := client.SendTransaction(ctx, signed); err != nil {
		return nil, nil, fmt.Errorf("send blob tx: %w", err)
	}

	receipt, err := bind.WaitMined(ctx, client, signed)
	if err != nil {
		return signed, nil, err
	}
	return signed, receipt, nil
}
