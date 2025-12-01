package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
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
			// Generate random recipient
			randomKey, _ := crypto.GenerateKey()
			toAddress := crypto.PubkeyToAddress(randomKey.PublicKey)

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
	trafficGenCmd.Flags().StringVar(&genRPCURL, "rpc", "http://localhost:8545", "RPC URL")
	trafficGenCmd.Flags().StringVar(&genPrivateKey, "key", "59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d", "Private key (hex)")
	trafficGenCmd.Flags().IntVar(&genTxCount, "count", 10, "Number of transactions to send")
}
