//go:build windows

package ipc

import (
	"net"
	"time"
)

// dial creates a TCP connection on Windows
func dial(path string, timeout time.Duration) (net.Conn, error) {
	// On Windows, we use TCP instead of Unix sockets
	// The path parameter is ignored, we use the same port as the server
	return net.DialTimeout("tcp", windowsIPCPort, timeout)
}
