//go:build windows

package ipc

import (
	"fmt"
	"net"
	"time"
)

// dial creates a TCP connection on Windows
// It first tries the port from the discovery file, then all known ports
func dial(path string, timeout time.Duration) (net.Conn, error) {
	// First, try the port from the discovery file
	discoveredPort := readPortFile()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", discoveredPort), timeout)
	if err == nil {
		return conn, nil
	}

	// If discovery file port failed, try all known ports
	for _, port := range windowsIPCPorts {
		if port == discoveredPort {
			continue // Already tried this one
		}
		conn, err = net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), timeout/time.Duration(len(windowsIPCPorts)))
		if err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("cannot connect to IPC server on any port")
}
