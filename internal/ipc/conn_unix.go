//go:build !windows

package ipc

import (
	"net"
	"time"
)

// dial creates a platform-specific connection
func dial(path string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", path, timeout)
}
