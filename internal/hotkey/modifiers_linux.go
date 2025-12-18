//go:build linux && !nogui

package hotkey

import "golang.design/x/hotkey"

// Platform-specific modifier mappings for Linux
// Linux uses Mod1 for Alt, Mod4 for Super/Win
var (
	modCmd    = hotkey.ModCtrl  // Cmd -> Ctrl on Linux
	modAlt    = hotkey.Mod1     // Option -> Mod1 (Alt) on Linux
	modCtrl   = hotkey.ModCtrl
	modShift  = hotkey.ModShift
)
