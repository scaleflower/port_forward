//go:build nogui

package main

import (
	"fmt"
	"log"
	"os"

	// Import engine to register protocols
	_ "pfm/internal/engine"

	"pfm/internal/cli"
	"pfm/internal/daemon"
)

func main() {
	args := os.Args[1:]

	// Check if running as service
	if cli.IsServiceRunCommand(args) {
		runAsService()
		return
	}

	// CLI mode only (no GUI available)
	if len(args) == 0 {
		// No args - show help instead of trying to start GUI
		cli.Run([]string{"help"})
		return
	}

	if err := cli.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runAsService runs the application as a background service
func runAsService() {
	log.Println("[Main] Starting in service mode...")

	d, err := daemon.New()
	if err != nil {
		log.Fatalf("[Main] Failed to create daemon: %v", err)
	}

	if err := d.Run(); err != nil {
		log.Fatalf("[Main] Service error: %v", err)
	}
}
