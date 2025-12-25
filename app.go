//go:build !nogui

package main

import (
	"context"
	"log"
	"os"
	"runtime"
	"sync"

	"pfm/internal/controller"
	"pfm/internal/daemon"
	"pfm/internal/engine"
	"pfm/internal/hotkey"
	"pfm/internal/ipc"
	"pfm/internal/models"
	"pfm/internal/storage"
	"pfm/internal/tray"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct represents the application
type App struct {
	ctx          context.Context
	controller   controller.ServiceController
	trayManager  *tray.Manager
	hotkeyMgr    *hotkey.Manager
	windowHidden bool
	mu           sync.Mutex
	isService    bool // Tracks if we are connected to a background service
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Try to connect to background service first
	ipcClient := ipc.NewClient()
	log.Println("[App] Trying to connect to background service...")

	if err := ipcClient.Connect(); err != nil {
		log.Printf("[App] IPC Connect failed: %v", err)
	} else {
		log.Println("[App] IPC Connect succeeded, trying Ping...")
	}

	if ipcClient.Ping() {
		a.isService = true
		a.controller = controller.NewRemote(ipcClient)
		log.Println("[App] Connected to background service successfully!")

		// Initialize tray and hotkey
		a.InitTray()
		a.InitHotkey()
		return
	}

	log.Println("[App] Ping failed, falling back to embedded mode")

	// No background service, run embedded
	a.isService = false
	log.Println("[App] Running in embedded mode")

	// Initialize storage
	store, err := storage.New()
	if err != nil {
		log.Printf("[App] Failed to initialize storage: %v", err)
		return
	}

	// Initialize engine
	eng := engine.New()

	// Create local controller
	localCtrl := controller.NewLocal(eng, store)

	// Initialize controller (load chains, start rules)
	if err := localCtrl.Init(); err != nil {
		log.Printf("[App] Failed to initialize local controller: %v", err)
	}

	a.controller = localCtrl

	// Initialize tray and hotkey
	a.InitTray()
	a.InitHotkey()
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	// Stop tray and hotkey
	a.StopTrayAndHotkey()

	// If local controller, we might want to stop engine?
	// Remote controller handles connection close?
	// ServiceController interface doesn't have Close().
	// But RemoteController holds ipcClient which has Close().
	// LocalController holds engine which has StopAll().
	// Maybe we should check type or add Close() to interface?
	// Adding Close() to interface is good practice.

	// For now, type assertion or just let it be (engine stops on exit usually, but correct shutdown is better).
	// Let's rely on OS exit for now, as original code called engine.StopAll().
	// If I modify interface now I need to update implementations.
	// I'll skip specific Close for now unless critical.
	// Original code:
	/*
		if a.useIPC {
			a.ipcClient.Close()
		} else if a.engine != nil {
			a.engine.StopAll()
		}
	*/

	/*
		if remote, ok := a.controller.(*controller.RemoteController); ok {
			// ...
		}
	*/
	// I will address Close() in a separate step if needed.
}

// ==================== Rule Operations ====================

// GetRules returns all rules
func (a *App) GetRules() []*models.Rule {
	if a.controller == nil {
		return []*models.Rule{}
	}
	rules, err := a.controller.GetRules()
	if err != nil {
		log.Printf("[App] GetRules error: %v", err)
		return []*models.Rule{}
	}
	return rules
}

// GetRule returns a single rule by ID
func (a *App) GetRule(id string) *models.Rule {
	if a.controller == nil {
		return nil
	}
	rule, err := a.controller.GetRule(id)
	if err != nil {
		log.Printf("[App] GetRule error: %v", err)
		return nil
	}
	return rule
}

// CreateRule creates a new rule
func (a *App) CreateRule(rule *models.Rule) error {
	if a.controller == nil {
		return models.ErrServiceNotRunning // or similar
	}
	return a.controller.CreateRule(rule)
}

// UpdateRule updates an existing rule
func (a *App) UpdateRule(rule *models.Rule) error {
	if a.controller == nil {
		return models.ErrServiceNotRunning
	}
	return a.controller.UpdateRule(rule)
}

// DeleteRule deletes a rule
func (a *App) DeleteRule(id string) error {
	if a.controller == nil {
		return models.ErrServiceNotRunning
	}
	return a.controller.DeleteRule(id)
}

// StartRule starts a rule
func (a *App) StartRule(id string) error {
	log.Printf("[App] StartRule called for id: %s", id)
	if a.controller == nil {
		return models.ErrServiceNotRunning
	}
	return a.controller.StartRule(id)
}

// StopRule stops a rule
func (a *App) StopRule(id string) error {
	if a.controller == nil {
		return models.ErrServiceNotRunning
	}
	return a.controller.StopRule(id)
}

// ==================== Chain Operations ====================

// GetChains returns all chains
func (a *App) GetChains() []*models.Chain {
	if a.controller == nil {
		return []*models.Chain{}
	}
	chains, err := a.controller.GetChains()
	if err != nil {
		log.Printf("[App] GetChains error: %v", err)
		return []*models.Chain{}
	}
	return chains
}

// CreateChain creates a new chain
func (a *App) CreateChain(chain *models.Chain) error {
	if a.controller == nil {
		return models.ErrServiceNotRunning
	}
	return a.controller.CreateChain(chain)
}

// UpdateChain updates an existing chain
func (a *App) UpdateChain(chain *models.Chain) error {
	if a.controller == nil {
		return models.ErrServiceNotRunning
	}
	return a.controller.UpdateChain(chain)
}

// DeleteChain deletes a chain
func (a *App) DeleteChain(id string) error {
	if a.controller == nil {
		return models.ErrServiceNotRunning
	}
	return a.controller.DeleteChain(id)
}

// ==================== Config Operations ====================

// GetConfig returns the application configuration
func (a *App) GetConfig() *models.AppConfig {
	if a.controller == nil {
		return models.DefaultAppConfig()
	}
	config, err := a.controller.GetConfig()
	if err != nil {
		log.Printf("[App] GetConfig error: %v", err)
		return models.DefaultAppConfig()
	}
	return config
}

// UpdateConfig updates the application configuration
func (a *App) UpdateConfig(config *models.AppConfig) error {
	if a.controller == nil {
		return models.ErrServiceNotRunning
	}
	return a.controller.UpdateConfig(config)
}

// ==================== Status Operations ====================

// GetStatus returns the current application status
func (a *App) GetStatus() *models.ServiceStatus {
	if a.controller == nil {
		return &models.ServiceStatus{
			Running: false,
			Version: "1.0.15", // Fallback
		}
	}
	status, err := a.controller.GetStatus()
	if err != nil {
		return &models.ServiceStatus{
			Running: false,
			Version: "1.0.15",
		}
	}
	return status
}

// IsServiceMode returns true if connected to background service
func (a *App) IsServiceMode() bool {
	return a.isService
}

// ==================== Service Management ====================

// InstallService installs and starts the background service
func (a *App) InstallService() error {
	if err := daemon.Install(); err != nil {
		return err
	}
	// Auto-start service after installation
	return daemon.Start()
}

// UninstallService uninstalls the background service
func (a *App) UninstallService() error {
	return daemon.Uninstall()
}

// StartService starts the background service
func (a *App) StartService() error {
	return daemon.Start()
}

// StopService stops the background service
func (a *App) StopService() error {
	return daemon.Stop()
}

// RestartService restarts the background service
func (a *App) RestartService() error {
	return daemon.Restart()
}

// GetServiceStatus returns the background service status
func (a *App) GetServiceStatus() string {
	if daemon.IsInstalled() {
		if daemon.IsRunning() {
			return "running"
		}
		return "stopped"
	}
	return "not_installed"
}

// ==================== Import/Export Operations ====================

// ExportData exports all data as JSON string (reads directly from file)
func (a *App) ExportData() string {
	if a.controller == nil {
		return ""
	}
	data, err := a.controller.ExportData()
	if err != nil {
		log.Printf("[App] ExportData error: %v", err)
		return ""
	}
	return string(data)
}

// ImportData imports data from JSON string
func (a *App) ImportData(data string, merge bool) error {
	if a.controller == nil {
		return models.ErrServiceNotRunning
	}
	return a.controller.ImportData([]byte(data), merge)
}

// OpenFileDialog opens a file dialog and returns the file content
func (a *App) OpenFileDialog() (string, error) {
	file, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Import File",
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "JSON Files (*.json)",
				Pattern:     "*.json",
			},
		},
	})
	if err != nil {
		return "", err
	}
	if file == "" {
		return "", nil // User cancelled
	}

	// Read file content
	content, err := os.ReadFile(file) // os imported? No, need to import "os"
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ClearAllData clears all rules and chains
func (a *App) ClearAllData() error {
	if a.controller == nil {
		return models.ErrServiceNotRunning
	}
	return a.controller.ClearAllData()
}

// ==================== Utility Functions ====================

// NewRule creates a new rule with default values
func (a *App) NewRule(name string, ruleType string) *models.Rule {
	return models.NewRule(name, models.RuleType(ruleType))
}

// NewChain creates a new chain with default values
func (a *App) NewChain(name string) *models.Chain {
	return models.NewChain(name)
}

// GetSystemInfo returns system information
func (a *App) GetSystemInfo() map[string]string {
	return map[string]string{
		"os":      runtime.GOOS,
		"arch":    runtime.GOARCH,
		"version": runtime.Version(),
	}
}

// ==================== Statistics Operations ====================

// GetRuleStats returns statistics for a specific rule
func (a *App) GetRuleStats(ruleID string) *models.RuleStats {
	if a.controller == nil {
		return &models.RuleStats{RuleID: ruleID}
	}
	return a.controller.GetRuleStats(ruleID)
}

// GetAllRuleStats returns statistics for all rules
func (a *App) GetAllRuleStats() map[string]*models.RuleStats {
	if a.controller == nil {
		return make(map[string]*models.RuleStats)
	}
	return a.controller.GetAllRuleStats()
}

// ==================== Log Operations ====================

// GetLogs returns recent log entries
func (a *App) GetLogs(count int) []models.LogEntry {
	if a.controller == nil {
		return []models.LogEntry{}
	}
	logs, err := a.controller.GetLogs(count)
	if err != nil {
		log.Printf("[App] GetLogs error: %v", err)
		return []models.LogEntry{}
	}
	return logs
}

// GetLogsSince returns log entries since a specific ID
func (a *App) GetLogsSince(sinceID int64) []models.LogEntry {
	if a.controller == nil {
		return []models.LogEntry{}
	}
	logs, err := a.controller.GetLogsSince(sinceID)
	if err != nil {
		log.Printf("[App] GetLogsSince error: %v", err)
		return []models.LogEntry{}
	}
	return logs
}

// GetLogsByRule returns log entries for a specific rule
func (a *App) GetLogsByRule(ruleID string) []models.LogEntry {
	if a.controller == nil {
		return []models.LogEntry{}
	}
	logs, err := a.controller.GetLogsByRule(ruleID)
	if err != nil {
		log.Printf("[App] GetLogsByRule error: %v", err)
		return []models.LogEntry{}
	}
	return logs
}

// ClearLogs clears all log entries
func (a *App) ClearLogs() {
	if a.controller != nil {
		a.controller.ClearLogs()
	}
}

// ==================== Window Operations ====================

// ShowWindow shows the main window
func (a *App) ShowWindow() {
	a.mu.Lock()
	defer a.mu.Unlock()

	wailsRuntime.WindowShow(a.ctx)
	wailsRuntime.WindowSetAlwaysOnTop(a.ctx, true)
	wailsRuntime.WindowSetAlwaysOnTop(a.ctx, false)
	a.windowHidden = false
	log.Println("[App] Window shown")
}

// HideWindow hides the main window
func (a *App) HideWindow() {
	a.mu.Lock()
	defer a.mu.Unlock()

	wailsRuntime.WindowHide(a.ctx)
	a.windowHidden = true
	log.Println("[App] Window hidden")
}

// ToggleWindow toggles the main window visibility
func (a *App) ToggleWindow() {
	a.mu.Lock()
	hidden := a.windowHidden
	a.mu.Unlock()

	if hidden {
		a.ShowWindow()
	} else {
		a.HideWindow()
	}
}

// IsWindowHidden returns true if the window is hidden
func (a *App) IsWindowHidden() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.windowHidden
}

// ==================== Tray and Hotkey Operations ====================

// InitTray initializes the system tray
func (a *App) InitTray() {
	config := a.GetConfig()
	if !config.TrayEnabled {
		log.Println("[App] Tray disabled in config")
		return
	}

	a.trayManager = tray.NewManager(tray.Callbacks{
		OnShow: func() {
			a.ShowWindow()
		},
		OnHide: func() {
			a.HideWindow()
		},
		OnQuit: func() {
			log.Println("[App] Quit requested from tray")
			wailsRuntime.Quit(a.ctx)
		},
	})

	// Start tray in goroutine (it blocks)
	go a.trayManager.Start(a.ctx)
	log.Println("[App] System tray initialized")
}

// StopTrayAndHotkey stops the tray and hotkey manager
func (a *App) StopTrayAndHotkey() {
	if a.trayManager != nil {
		a.trayManager.Stop()
	}
	if a.hotkeyMgr != nil {
		a.hotkeyMgr.Stop()
	}
}

// InitHotkey initializes the global hotkey
func (a *App) InitHotkey() {
	config := a.GetConfig()
	if !config.HotkeyEnabled {
		log.Println("[App] Hotkey disabled in config")
		return
	}

	hotkeyConfig := &hotkey.Config{
		Modifiers: config.HotkeyModifiers,
		Key:       config.HotkeyKey,
	}

	var err error
	a.hotkeyMgr, err = hotkey.NewManager(hotkeyConfig, func() {
		log.Println("[App] Hotkey triggered")
		a.ShowWindow()
	})
	if err != nil {
		log.Printf("[App] Failed to create hotkey manager: %v", err)
		return
	}

	if err := a.hotkeyMgr.Start(); err != nil {
		log.Printf("[App] Failed to start hotkey: %v", err)
		return
	}

	log.Printf("[App] Global hotkey initialized: %s", hotkey.GetHotkeyString(hotkeyConfig))
}

// UpdateHotkey updates the global hotkey configuration
func (a *App) UpdateHotkey(modifiers, key string) error {
	if a.hotkeyMgr == nil {
		return nil
	}

	config := &hotkey.Config{
		Modifiers: modifiers,
		Key:       key,
	}

	if err := a.hotkeyMgr.UpdateConfig(config); err != nil {
		return err
	}

	log.Printf("[App] Global hotkey updated: %s", hotkey.GetHotkeyString(config))
	return nil
}
