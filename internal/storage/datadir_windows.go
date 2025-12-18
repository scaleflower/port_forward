//go:build windows

package storage

import (
	"os"
	"path/filepath"
)

// getDefaultDataDir returns the default data directory for Windows
// Uses C:\ProgramData\pfm for service compatibility (SYSTEM account access)
func getDefaultDataDir() string {
	// Use ProgramData for shared access between user and service
	programData := os.Getenv("ProgramData")
	if programData == "" {
		programData = `C:\ProgramData`
	}
	return filepath.Join(programData, "pfm")
}
