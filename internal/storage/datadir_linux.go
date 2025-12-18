//go:build linux

package storage

import (
	"os"
	"path/filepath"
)

// getDefaultDataDir returns the default data directory for Linux
func getDefaultDataDir() string {
	// Check if running as root (service mode)
	if os.Getuid() == 0 {
		return "/var/lib/pfm"
	}

	// User mode: use ~/.config/pfm
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "/var/lib/pfm"
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "pfm")
}
