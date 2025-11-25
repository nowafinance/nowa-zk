package main

import (
	"log"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "prover",
	Short: "ZK Sequencer for Tan-ZK network",
	Long:  `A ZK Sequencer service that monitors the Tan-ZK blockchain, collects transactions, builds batches, and provides APIs.`,
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

