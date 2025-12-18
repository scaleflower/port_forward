//go:build darwin

package storage

import (
	"os"
	"path/filepath"
)

// getDefaultDataDir returns the default data directory for macOS
// Uses executable directory for portable deployment
func getDefaultDataDir() string {
	// Use executable's directory/data for portable deployment
	execPath, err := os.Executable()
	if err == nil {
		execPath, err = filepath.EvalSymlinks(execPath)
		if err == nil {
			// For macOS .app bundle, go up from Contents/MacOS to .app level
			execDir := filepath.Dir(execPath)
			if filepath.Base(execDir) == "MacOS" {
				// Inside .app bundle: use .app/../data
				appDir := filepath.Dir(filepath.Dir(execDir))
				return filepath.Join(filepath.Dir(appDir), "data")
			}
			return filepath.Join(execDir, "data")
		}
	}

	// Fallback to ~/Library/Application Support/pfm
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/var/root"
	}
	return filepath.Join(home, "Library", "Application Support", "pfm")
}
