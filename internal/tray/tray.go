package tray

import (
	"context"
	"log"
	"sync"
)

// Note: Due to symbol conflicts between Wails and fyne.io/systray on macOS,
// we implement a simplified tray manager that integrates with Wails' native menu.
// The actual system tray functionality is handled through HideWindowOnClose
// and the global hotkey feature.

// Callbacks defines the callback functions for tray events
type Callbacks struct {
	OnShow func()
	OnHide func()
	OnQuit func()
}

// Manager manages the window visibility state
// On macOS, instead of a separate tray icon, we use:
// 1. HideWindowOnClose in Wails options (window minimizes to dock on close)
// 2. Global hotkey to bring the window back
type Manager struct {
	mu        sync.Mutex
	ctx       context.Context
	callbacks Callbacks
	running   bool
}

// NewManager creates a new tray manager
func NewManager(callbacks Callbacks) *Manager {
	return &Manager{
		callbacks: callbacks,
	}
}

// Start starts the tray manager
// Note: On macOS with Wails, the dock icon serves as the "tray" icon
// Clicking the dock icon will show the window
func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.ctx = ctx
	m.mu.Unlock()

	log.Println("[Tray] Manager started (using Wails dock integration)")
}

// Stop stops the tray manager
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		m.running = false
		log.Println("[Tray] Manager stopped")
	}
}

// IsRunning returns whether the tray manager is running
func (m *Manager) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

// ShowWindow calls the show callback
func (m *Manager) ShowWindow() {
	if m.callbacks.OnShow != nil {
		m.callbacks.OnShow()
	}
}

// HideWindow calls the hide callback
func (m *Manager) HideWindow() {
	if m.callbacks.OnHide != nil {
		m.callbacks.OnHide()
	}
}

// Quit calls the quit callback
func (m *Manager) Quit() {
	if m.callbacks.OnQuit != nil {
		m.callbacks.OnQuit()
	}
}
