//go:build windows

package storage

import (
	"os"
	"path/filepath"
)

// getDefaultDataDir returns the default data directory for Windows
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

	// Fallback to ProgramData
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "pfm")
}
