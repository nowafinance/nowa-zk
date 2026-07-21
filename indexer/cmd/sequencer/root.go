package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/nowafinance/nowa-zk/sequencer/pkg/logger"
)

var rootCmd = &cobra.Command{
	Use:   "sequencer",
	Short: "ZK Sequencer for Nowa-ZK network",
	Long:  `A ZK Sequencer service that monitors the Nowa-ZK blockchain, collects transactions, builds batches, and provides APIs.`,
}

func init() {
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(trafficGenCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("Failed to execute command: %v", err)
		os.Exit(1)
	}
}
