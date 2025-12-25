package main

import (
	"fmt"
	"os"

	"pfm/internal/daemon"

	// Import engine to register protocols
	_ "pfm/internal/engine"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	var err error
	switch cmd {
	case "run":
		err = runService()
	case "install":
		err = daemon.Install()
		if err == nil {
			fmt.Println("Service installed successfully")
		}
	case "uninstall":
		err = daemon.Uninstall()
		if err == nil {
			fmt.Println("Service uninstalled successfully")
		}
	case "start":
		err = daemon.Start()
		if err == nil {
			fmt.Println("Service started successfully")
		}
	case "stop":
		err = daemon.Stop()
		if err == nil {
			fmt.Println("Service stopped successfully")
		}
	case "restart":
		err = daemon.Restart()
		if err == nil {
			fmt.Println("Service restarted successfully")
		}
	case "status":
		err = printStatus()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runService() error {
	d, err := daemon.New()
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}
	return d.Run()
}

func printStatus() error {
	if daemon.IsInstalled() {
		if daemon.IsRunning() {
			fmt.Println("Service status: Running")
		} else {
			fmt.Println("Service status: Stopped")
		}
	} else {
		fmt.Println("Service status: Not installed")
	}
	return nil
}

func printUsage() {
	fmt.Print(`Port Forward Manager - Service

Usage:
  pfm-service <command>

Commands:
  run         Run the service in foreground (used by service manager)
  install     Install as a system service
  uninstall   Uninstall the system service
  start       Start the installed service
  stop        Stop the installed service
  restart     Restart the installed service
  status      Show service status
  help        Show this help message

Examples:
  pfm-service install    Install as system service
  pfm-service start      Start the service
  pfm-service status     Check service status
`)
}
