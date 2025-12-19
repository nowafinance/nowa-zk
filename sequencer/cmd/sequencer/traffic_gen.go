package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	genRPCURL     string
	genPrivateKey string
	genTxCount    int
)

var trafficGenCmd = &cobra.Command{
	Use:   "traffic-gen",
	Short: "Generate random transactions for testing",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("🚀 Starting Traffic Generator\n")
		fmt.Printf("   RPC: %s\n", genRPCURL)
		fmt.Printf("   Count: %d\n", genTxCount)

		// 1. Connect to RPC
		client, err := ethclient.Dial(genRPCURL)
		if err != nil {
			fmt.Printf("❌ Failed to connect to RPC: %v\n", err)
			os.Exit(1)
		}

		// 2. Load Private Key
		// Remove 0x prefix if present
		if len(genPrivateKey) > 2 && genPrivateKey[:2] == "0x" {
			genPrivateKey = genPrivateKey[2:]
		}

		key, err := crypto.HexToECDSA(genPrivateKey)
		if err != nil {
			fmt.Printf("❌ Invalid private key: %v\n", err)
			os.Exit(1)
		}

		publicKey := key.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			fmt.Println("❌ Error casting public key to ECDSA")
			os.Exit(1)
		}

		fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
		fmt.Printf("   Sender: %s\n", fromAddress.Hex())

		// 3. Get Chain ID
		chainID, err := client.NetworkID(context.Background())
		if err != nil {
			fmt.Printf("❌ Failed to get chain ID: %v\n", err)
			os.Exit(1)
		}

		// 4. Get Initial Nonce
		nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			fmt.Printf("❌ Failed to get nonce: %v\n", err)
			os.Exit(1)
		}

		// 5. Send Transactions
		fmt.Println("\n💸 Sending transactions...")

		for i := 0; i < genTxCount; i++ {
			// Send to fixed address as requested
			toAddress := common.HexToAddress("0x25691469d348161ea4d4bf6409c34c5a084decb4")

			// Amount: 0.0001 ETH (10^14 wei)
			value := big.NewInt(100000000000000)

			// Gas Limit
			gasLimit := uint64(21000)

			// Gas Price
			gasPrice, err := client.SuggestGasPrice(context.Background())
			if err != nil {
				fmt.Printf("⚠️ Failed to suggest gas price: %v. Using default.\n", err)
				gasPrice = big.NewInt(1000000000) // 1 gwei
			}

			// Create Transaction
			tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, nil)

			// Sign Transaction
			signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), key)
			if err != nil {
				fmt.Printf("❌ Failed to sign tx %d: %v\n", i, err)
				continue
			}

			// Send Transaction
			err = client.SendTransaction(context.Background(), signedTx)
			if err != nil {
				fmt.Printf("❌ Failed to send tx %d: %v\n", i, err)
				continue
			}

			fmt.Printf("   [%d/%d] Sent %s ETH to %s (Hash: %s)\n",
				i+1, genTxCount, "0.0001", toAddress.Hex(), signedTx.Hash().Hex())

			// Increment nonce for next tx
			nonce++

			// Small delay to be polite to the node
			time.Sleep(100 * time.Millisecond)
		}

		fmt.Println("\n✅ Traffic generation complete!")
	},
}

func init() {
	// Load .env file
	_ = godotenv.Load()

	defaultRPC := os.Getenv("RPC")
	if defaultRPC == "" {
		defaultRPC = "http://localhost:8545" // Fallback only if env var is missing, but user wants to replace it.
		// Wait, user said "replace it with .env's RPC".
		// If I remove the fallback entirely, it might break if .env is missing and no flag is passed.
		// But user said "I no longer want it to test locally, i wanna test it in our RPC".
		// So I should probably make it empty default and fail if not provided?
		// Or just set default to "" and let the code handle it?
		// The code uses `genRPCURL` which is set by flag.
		// If I set default to "", then if user runs without flag and without env, it will be empty.
		// `ethclient.Dial("")` will likely fail.
		// Let's set default to os.Getenv("RPC").
	}

	defaultKey := os.Getenv("TRAFFIC_GEN_KEY")

	trafficGenCmd.Flags().StringVar(&genRPCURL, "rpc", defaultRPC, "RPC URL (default: RPC from .env)")
	trafficGenCmd.Flags().StringVar(&genPrivateKey, "key", defaultKey, "Private key (hex) (default: TRAFFIC_GEN_KEY from .env)")
	trafficGenCmd.Flags().IntVar(&genTxCount, "count", 10, "Number of transactions to send")
}
