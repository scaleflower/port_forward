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

// getPortFilePath returns the path to the port discovery file
func getPortFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	return filepath.Join(configDir, "pfm", "ipc_port")
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
			writePortFile(port)
			log.Printf("[IPC] Successfully bound to port %d", port)
			return listener, nil
		}
		log.Printf("[IPC] Port %d unavailable: %v, trying next...", port, err)
	}

	// All ports failed
	return nil, fmt.Errorf("all IPC ports unavailable: %v", err)
}

// writePortFile writes the active port to a discovery file
func writePortFile(port int) {
	portFile := getPortFilePath()
	dir := filepath.Dir(portFile)
	os.MkdirAll(dir, 0755)
	os.WriteFile(portFile, []byte(strconv.Itoa(port)), 0644)
}

// readPortFile reads the active port from the discovery file
func readPortFile() int {
	portFile := getPortFilePath()
	data, err := os.ReadFile(portFile)
	if err != nil {
		return windowsIPCPorts[0] // Default to first port
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return windowsIPCPorts[0]
	}
	return port
}

// cleanupListener cleans up platform-specific resources
func cleanupListener(path string) {
	// Remove port file when server stops
	os.Remove(getPortFilePath())
}
