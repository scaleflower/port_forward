//go:build !nogui

package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	// Import engine to register protocols
	_ "pfm/internal/engine"

	"pfm/internal/cli"
	"pfm/internal/daemon"
	"pfm/internal/singleinstance"
)

//go:embed all:frontend/dist
var assets embed.FS

// Global variables for single instance and app
var (
	singleInst *singleinstance.Instance
	guiApp     *App
)

func main() {
	args := os.Args[1:]

	// Check if running as service
	if cli.IsServiceRunCommand(args) {
		runAsService()
		return
	}

	// Check if CLI command
	if cli.IsCLICommand(args) {
		if err := cli.Run(args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Normal GUI mode (no args or unrecognized args)
	runGUI()
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

// runGUI runs the application with GUI
func runGUI() {
	// Create single instance manager (callback will be set after app is created)
	singleInst = singleinstance.New("pfm-gui", nil)

	// Try to acquire single instance lock
	isFirst, err := singleInst.TryLock()
	if err != nil {
		log.Printf("[Main] Single instance lock error: %v", err)
		// Continue anyway, just log the error
	}

	if !isFirst {
		// Another instance is running, send wakeup signal and exit
		log.Println("[Main] Another instance is already running, sending wakeup signal...")
		if err := singleInst.SendWakeupSignal(); err != nil {
			log.Printf("[Main] Failed to send wakeup signal: %v", err)
		}
		// Give some time for the signal to be processed
		time.Sleep(500 * time.Millisecond)
		log.Println("[Main] Exiting duplicate instance")
		os.Exit(0)
	}

	log.Println("[Main] This is the first instance, starting GUI...")

	// Create an instance of the app structure
	guiApp = NewApp()

	// Set the wakeup callback to show the window
	singleInst.SetWakeupCallback(func() {
		log.Println("[Main] Wakeup callback triggered, showing window...")
		if guiApp != nil {
			guiApp.ShowWindow()
		}
	})

	// Start listening for wakeup signals from other instances
	if err := singleInst.StartWakeupListener(); err != nil {
		log.Printf("[Main] Failed to start wakeup listener: %v", err)
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:             "Port Forward Manager",
		Width:             1200,
		Height:            800,
		MinWidth:          800,
		MinHeight:         600,
		HideWindowOnClose: true, // Hide to tray instead of quit on close
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 245, G: 247, B: 250, A: 1},
		OnStartup:        guiApp.startup,
		OnShutdown: func(ctx context.Context) {
			// Clean up single instance before app shutdown
			if singleInst != nil {
				singleInst.Unlock()
			}
			guiApp.shutdown(ctx)
		},
		Bind: []interface{}{
			guiApp,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
