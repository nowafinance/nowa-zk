package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/tannetwork/zk-sequencer/sequencer/pkg/logger"
)

var rootCmd = &cobra.Command{
	Use:   "sequencer",
	Short: "ZK Sequencer for Tan-ZK network",
	Long:  `A ZK Sequencer service that monitors the Tan-ZK blockchain, collects transactions, builds batches, and provides APIs.`,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("Failed to execute command: %v", err)
		os.Exit(1)
	}
}

