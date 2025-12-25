//go:build windows

package ipc

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Windows IPC ports - try these in order if some are occupied
var windowsIPCPorts = []int{19846, 19856, 19866, 19876, 19886}

// activeIPCPort stores the port that's actually being used
var activeIPCPort int

// getPortFilePaths returns potential paths for the port discovery file
// Order: Shared (ProgramData), User (AppData)
func getPortFilePaths() []string {
	var paths []string

	// 1. Shared (ProgramData) - for Service
	if programData := os.Getenv("ProgramData"); programData != "" {
		paths = append(paths, filepath.Join(programData, "pfm", "ipc_port"))
	} else if allUsers := os.Getenv("ALLUSERSPROFILE"); allUsers != "" {
		paths = append(paths, filepath.Join(allUsers, "pfm", "ipc_port"))
	}

	// 2. User (AppData) - for Embedded/User mode
	if userConfig, err := os.UserConfigDir(); err == nil {
		paths = append(paths, filepath.Join(userConfig, "pfm", "ipc_port"))
	}

	return paths
}

// createListener creates a TCP listener on Windows, trying multiple ports
func createListener(path string) (net.Listener, error) {
	var listener net.Listener
	var err error

	// Try multiple ports
	for _, port := range windowsIPCPorts {
		addr := fmt.Sprintf("127.0.0.1:%d", port)
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			activeIPCPort = port
			// Write port to file for client discovery
			if err := writePortFile(port); err != nil {
				log.Printf("[IPC] Warning: failed to write port file: %v", err)
			}
			log.Printf("[IPC] Successfully bound to port %d", port)
			return listener, nil
		}
		log.Printf("[IPC] Port %d unavailable: %v, trying next...", port, err)
	}

	// All ports failed
	return nil, fmt.Errorf("all IPC ports unavailable: %v", err)
}

// writePortFile writes the active port to discovery file(s)
func writePortFile(port int) error {
	paths := getPortFilePaths()
	var lastErr error
	written := false

	data := []byte(strconv.Itoa(port))

	for _, path := range paths {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			lastErr = err
			continue
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			lastErr = err
			continue
		}
		written = true
	}

	if !written && lastErr != nil {
		return fmt.Errorf("failed to write to any port file: %v", lastErr)
	}
	return nil
}

// readPortFile reads the active port from the discovery file
// (Used by client in conn_windows.go, but defined here for package consistency)
// NOTE: conn_windows.go implements its own logic using getPortFilePaths now
func readPortFile() int {
	paths := getPortFilePaths()
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			if port, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				return port
			}
		}
	}
	return windowsIPCPorts[0]
}

// cleanupListener cleans up platform-specific resources
func cleanupListener(path string) {
	// Remove port files
	paths := getPortFilePaths()
	for _, p := range paths {
		os.Remove(p)
	}
}
