//go:build linux

package storage

import (
	"os"
	"path/filepath"
)

// getDefaultDataDir returns the default data directory for Linux
// Uses executable directory for portable deployment
func getDefaultDataDir() string {
	// Use executable's directory/data for portable deployment
	execPath, err := os.Executable()
	if err == nil {
		execPath, err = filepath.EvalSymlinks(execPath)
		if err == nil {
			return filepath.Join(filepath.Dir(execPath), "data")
		}
	}

	// Fallback based on user
	if os.Getuid() == 0 {
		return "/var/lib/pfm"
	}

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
