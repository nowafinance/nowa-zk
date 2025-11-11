package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tannetwork/zk-sequencer/sequencer/internal/sequencer"
)

func main() {
	// Create sequencer service
	s := sequencer.New()

	// Start the service
	if err := s.Start(); err != nil {
		log.Fatalf("Failed to start sequencer: %v", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("\n🛑 Shutdown signal received...")

	// Stop the service gracefully
	if err := s.Stop(); err != nil {
		log.Fatalf("Failed to stop sequencer: %v", err)
	}
}


