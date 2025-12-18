//go:build linux

package hotkey

import "golang.design/x/hotkey"

// Platform-specific modifier mappings for Linux
// Linux doesn't have Cmd/Option, map to Ctrl/Alt
var (
	modCmd    = hotkey.ModCtrl // Cmd -> Ctrl on Linux
	modAlt    = hotkey.ModAlt  // Option -> Alt on Linux
	modCtrl   = hotkey.ModCtrl
	modShift  = hotkey.ModShift
)
