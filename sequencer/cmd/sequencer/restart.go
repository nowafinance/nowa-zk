package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/tannetwork/zk-sequencer/sequencer/internal/sequencer"
	"github.com/tannetwork/zk-sequencer/sequencer/pkg/config"
	"github.com/tannetwork/zk-sequencer/sequencer/pkg/logger"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the sequencer service",
	Long:  `Stop the sequencer if running and start it again with optional configuration overrides.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("🔄 Restarting ZK Sequencer...")

		// Load configuration from env and command-line flags
		cfg, err := config.LoadWithOverrides(rpcURL, wsURL, stateDBPath, batchSize, apiPort)
		if err != nil {
			logger.Error("Failed to load configuration: %v", err)
			return err
		}

		// Create sequencer service with config
		s := sequencer.NewWithConfig(cfg)

		// Stop any existing instance (if running)
		logger.Info("Stopping any existing sequencer instance...")
		_ = s.Stop()
		time.Sleep(1 * time.Second) // Brief pause

		// Start the service
		if err := s.Start(); err != nil {
			logger.Error("Failed to start sequencer: %v", err)
			return err
		}

		logger.Info("✅ ZK Sequencer restarted successfully")

		// Wait for interrupt signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		<-sigChan
		logger.Info("\n🛑 Shutdown signal received...")

		// Stop the service gracefully
		if err := s.Stop(); err != nil {
			logger.Error("Failed to stop sequencer: %v", err)
			return err
		}

		logger.Info("✅ ZK Sequencer stopped gracefully")
		return nil
	},
}

func init() {
	restartCmd.Flags().StringVar(&rpcURL, "rpc-url", "", "Tan-ZK RPC URL (overrides TAN_ZK_RPC_URL env var)")
	restartCmd.Flags().StringVar(&wsURL, "ws-url", "", "Tan-ZK WebSocket URL (overrides TAN_ZK_WS_URL env var)")
	restartCmd.Flags().StringVar(&stateDBPath, "state-db-path", "", "Path to state database (overrides STATE_DB_PATH env var)")
	restartCmd.Flags().IntVar(&batchSize, "batch-size", 0, "Batch size (overrides BATCH_SIZE env var)")
	restartCmd.Flags().IntVar(&apiPort, "api-port", 0, "API server port (overrides API_PORT env var)")
}

