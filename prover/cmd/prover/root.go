package main

import (
	"log"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "prover",
	Short: "ZK Indexer for Nowa-ZK network",
	Long:  `A ZK Indexer service that monitors the Nowa-ZK blockchain, collects transactions, builds batches, and provides APIs.`,
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(setupCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute command: %v", err)
	}
}
