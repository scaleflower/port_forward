//go:build !windows

package ipc

import (
	"net"
	"os"
)

// createListener creates a platform-specific listener
func createListener(path string) (net.Listener, error) {
	// Remove existing socket file
	os.Remove(path)

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	// Set permissions on socket - allow all users to connect
	os.Chmod(path, 0666)

	return listener, nil
}

// cleanupListener cleans up platform-specific resources
func cleanupListener(path string) {
	os.Remove(path)
}
