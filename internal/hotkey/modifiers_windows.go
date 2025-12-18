//go:build windows && !nogui

package hotkey

import "golang.design/x/hotkey"

// Platform-specific modifier mappings for Windows
// Windows doesn't have Cmd/Option, map to Ctrl/Alt
var (
	modCmd    = hotkey.ModCtrl // Cmd -> Ctrl on Windows
	modAlt    = hotkey.ModAlt  // Option -> Alt on Windows
	modCtrl   = hotkey.ModCtrl
	modShift  = hotkey.ModShift
)
