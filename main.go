package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	// Import engine to register protocols
	_ "pfm/internal/engine"

	"pfm/internal/daemon"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Check if running as service
	if len(os.Args) >= 3 && os.Args[1] == "service" && os.Args[2] == "run" {
		runAsService()
		return
	}

	// Normal GUI mode
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
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
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
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
