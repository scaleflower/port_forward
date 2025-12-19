//go:build windows

package ipc

import (
	"net"
)

// Windows IPC port - used for service communication
const windowsIPCPort = "127.0.0.1:19846"

// createListener creates a TCP listener on Windows
func createListener(path string) (net.Listener, error) {
	// On Windows, we use TCP instead of Unix sockets/named pipes
	// The path parameter is ignored, we use a fixed port
	return net.Listen("tcp", windowsIPCPort)
}

// cleanupListener cleans up platform-specific resources
func cleanupListener(path string) {
	// Nothing to clean up for TCP listener on Windows
}
