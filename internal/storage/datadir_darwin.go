//go:build darwin

package storage

import (
	"os"
	"path/filepath"
)

// getDefaultDataDir returns the default data directory for macOS
func getDefaultDataDir() string {
	// Use ~/Library/Application Support/pfm
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/var/root"
	}
	return filepath.Join(home, "Library", "Application Support", "pfm")
}
