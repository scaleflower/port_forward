//go:build !windows

package daemon

// isAdmin checks if the current process has administrator privileges
// On non-Windows platforms, this always returns true (permission handled differently)
func isAdmin() bool {
	return true
}
