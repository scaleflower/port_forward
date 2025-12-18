//go:build darwin && !nogui

package hotkey

import "golang.design/x/hotkey"

// Platform-specific modifier mappings for macOS
var (
	modCmd    = hotkey.ModCmd
	modAlt    = hotkey.ModOption
	modCtrl   = hotkey.ModCtrl
	modShift  = hotkey.ModShift
)
