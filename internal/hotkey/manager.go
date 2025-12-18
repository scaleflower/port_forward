package hotkey

import (
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"

	"golang.design/x/hotkey"
)

// Manager manages global hotkeys
type Manager struct {
	mu       sync.Mutex
	hk       *hotkey.Hotkey
	callback func()
	running  bool
	stopCh   chan struct{}
}

// Config represents hotkey configuration
type Config struct {
	Modifiers string // e.g., "cmd+shift", "ctrl+shift", "cmd+alt"
	Key       string // e.g., "p", "f", "m"
}

// DefaultConfig returns the default hotkey configuration
func DefaultConfig() *Config {
	// Use Cmd on macOS, Ctrl on Windows/Linux
	mods := "ctrl+shift"
	if runtime.GOOS == "darwin" {
		mods = "cmd+shift"
	}
	return &Config{
		Modifiers: mods,
		Key:       "p",
	}
}

// NewManager creates a new hotkey manager
func NewManager(cfg *Config, callback func()) (*Manager, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	mods, err := parseModifiers(cfg.Modifiers)
	if err != nil {
		return nil, fmt.Errorf("invalid modifiers: %w", err)
	}

	key, err := parseKey(cfg.Key)
	if err != nil {
		return nil, fmt.Errorf("invalid key: %w", err)
	}

	hk := hotkey.New(mods, key)

	return &Manager{
		hk:       hk,
		callback: callback,
		stopCh:   make(chan struct{}),
	}, nil
}

// Start registers and starts listening for the hotkey
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}

	log.Println("[Hotkey] Registering hotkey...")
	if err := m.hk.Register(); err != nil {
		m.mu.Unlock()
		log.Printf("[Hotkey] Registration failed: %v", err)
		log.Println("[Hotkey] NOTE: On macOS, you need to grant Accessibility permission:")
		log.Println("[Hotkey]   System Settings → Privacy & Security → Accessibility → Enable for pfm")
		return fmt.Errorf("failed to register hotkey (check Accessibility permission): %w", err)
	}

	log.Println("[Hotkey] Hotkey registered successfully")
	m.running = true
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	// Start listening in a goroutine
	go m.listen()

	return nil
}

// listen waits for hotkey events
func (m *Manager) listen() {
	log.Println("[Hotkey] Started listening for hotkey events...")
	for {
		select {
		case <-m.stopCh:
			log.Println("[Hotkey] Stopped listening")
			return
		case <-m.hk.Keydown():
			log.Println("[Hotkey] Hotkey triggered!")
			if m.callback != nil {
				m.callback()
			}
		}
	}
}

// Stop unregisters and stops listening for the hotkey
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	close(m.stopCh)
	m.running = false

	if err := m.hk.Unregister(); err != nil {
		return fmt.Errorf("failed to unregister hotkey: %w", err)
	}

	return nil
}

// IsRunning returns whether the hotkey manager is running
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// UpdateConfig updates the hotkey configuration
// This will unregister the old hotkey and register the new one
func (m *Manager) UpdateConfig(cfg *Config) error {
	m.mu.Lock()
	wasRunning := m.running
	m.mu.Unlock()

	if wasRunning {
		if err := m.Stop(); err != nil {
			log.Printf("[Hotkey] Warning: failed to stop old hotkey: %v", err)
		}
	}

	mods, err := parseModifiers(cfg.Modifiers)
	if err != nil {
		return fmt.Errorf("invalid modifiers: %w", err)
	}

	key, err := parseKey(cfg.Key)
	if err != nil {
		return fmt.Errorf("invalid key: %w", err)
	}

	m.mu.Lock()
	m.hk = hotkey.New(mods, key)
	m.mu.Unlock()

	if wasRunning {
		return m.Start()
	}

	return nil
}

// parseModifiers parses a modifier string into hotkey modifiers
func parseModifiers(s string) ([]hotkey.Modifier, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	parts := strings.Split(s, "+")

	var mods []hotkey.Modifier
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch part {
		case "cmd", "command", "super":
			// Use platform-specific modCmd (Cmd on macOS, Ctrl on Windows/Linux)
			mods = append(mods, modCmd)
		case "ctrl", "control":
			mods = append(mods, modCtrl)
		case "shift":
			mods = append(mods, modShift)
		case "alt", "option":
			// Use platform-specific modAlt (Option on macOS, Alt on Windows/Linux)
			mods = append(mods, modAlt)
		case "":
			// Skip empty parts
		default:
			return nil, fmt.Errorf("unknown modifier: %s", part)
		}
	}

	if len(mods) == 0 {
		return nil, fmt.Errorf("no modifiers specified")
	}

	return mods, nil
}

// parseKey parses a key string into a hotkey key
func parseKey(s string) (hotkey.Key, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	// Map single characters to keys
	keyMap := map[string]hotkey.Key{
		"a": hotkey.KeyA,
		"b": hotkey.KeyB,
		"c": hotkey.KeyC,
		"d": hotkey.KeyD,
		"e": hotkey.KeyE,
		"f": hotkey.KeyF,
		"g": hotkey.KeyG,
		"h": hotkey.KeyH,
		"i": hotkey.KeyI,
		"j": hotkey.KeyJ,
		"k": hotkey.KeyK,
		"l": hotkey.KeyL,
		"m": hotkey.KeyM,
		"n": hotkey.KeyN,
		"o": hotkey.KeyO,
		"p": hotkey.KeyP,
		"q": hotkey.KeyQ,
		"r": hotkey.KeyR,
		"s": hotkey.KeyS,
		"t": hotkey.KeyT,
		"u": hotkey.KeyU,
		"v": hotkey.KeyV,
		"w": hotkey.KeyW,
		"x": hotkey.KeyX,
		"y": hotkey.KeyY,
		"z": hotkey.KeyZ,
		"0": hotkey.Key0,
		"1": hotkey.Key1,
		"2": hotkey.Key2,
		"3": hotkey.Key3,
		"4": hotkey.Key4,
		"5": hotkey.Key5,
		"6": hotkey.Key6,
		"7": hotkey.Key7,
		"8": hotkey.Key8,
		"9": hotkey.Key9,
		"space":  hotkey.KeySpace,
		"return": hotkey.KeyReturn,
		"enter":  hotkey.KeyReturn,
		"escape": hotkey.KeyEscape,
		"esc":    hotkey.KeyEscape,
		"tab":    hotkey.KeyTab,
		"delete": hotkey.KeyDelete,
		"f1":     hotkey.KeyF1,
		"f2":     hotkey.KeyF2,
		"f3":     hotkey.KeyF3,
		"f4":     hotkey.KeyF4,
		"f5":     hotkey.KeyF5,
		"f6":     hotkey.KeyF6,
		"f7":     hotkey.KeyF7,
		"f8":     hotkey.KeyF8,
		"f9":     hotkey.KeyF9,
		"f10":    hotkey.KeyF10,
		"f11":    hotkey.KeyF11,
		"f12":    hotkey.KeyF12,
	}

	if key, ok := keyMap[s]; ok {
		return key, nil
	}

	return 0, fmt.Errorf("unknown key: %s", s)
}

// GetHotkeyString returns a human-readable string of the configured hotkey
func GetHotkeyString(cfg *Config) string {
	mods := strings.ToUpper(cfg.Modifiers)
	mods = strings.ReplaceAll(mods, "+", " + ")
	key := strings.ToUpper(cfg.Key)
	return fmt.Sprintf("%s + %s", mods, key)
}
