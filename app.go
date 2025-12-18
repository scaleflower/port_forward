package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"runtime"
	"sync"

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
	engine       *engine.Engine
	store        *storage.Store
	ipcClient    *ipc.Client
	useIPC       bool // true if connecting to background service
	trayManager  *tray.Manager
	hotkeyMgr    *hotkey.Manager
	windowHidden bool
	mu           sync.Mutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Try to connect to background service first
	a.ipcClient = ipc.NewClient()
	log.Println("[App] Trying to connect to background service...")

	if err := a.ipcClient.Connect(); err != nil {
		log.Printf("[App] IPC Connect failed: %v", err)
	} else {
		log.Println("[App] IPC Connect succeeded, trying Ping...")
	}

	if a.ipcClient.Ping() {
		a.useIPC = true
		log.Println("[App] Connected to background service successfully!")
		// Initialize tray and hotkey
		a.InitTray()
		a.InitHotkey()
		return
	}

	log.Println("[App] Ping failed, falling back to embedded mode")

	// No background service, run embedded
	a.useIPC = false
	log.Println("[App] Running in embedded mode")

	// Initialize storage
	store, err := storage.New()
	if err != nil {
		log.Printf("[App] Failed to initialize storage: %v", err)
		return
	}
	a.store = store

	// Initialize engine
	a.engine = engine.New()
	a.engine.SetChains(a.store.GetChains())

	// Set status change callback to sync engine errors to store
	a.engine.SetStatusChangeCallback(func(ruleID string, status string, errorMsg string) {
		log.Printf("[App] Status change callback: rule=%s, status=%s, error=%s", ruleID, status, errorMsg)
		if status == "error" {
			a.store.UpdateRuleStatus(ruleID, models.RuleStatusError, errorMsg)
		} else if status == "stopped" {
			a.store.UpdateRuleStatus(ruleID, models.RuleStatusStopped, "")
		}
	})

	// Start enabled rules and sync status
	for _, rule := range a.store.GetRules() {
		if rule.Enabled {
			if err := a.engine.StartRule(rule); err != nil {
				log.Printf("[App] Failed to start rule %s: %v", rule.Name, err)
				a.store.UpdateRuleStatus(rule.ID, models.RuleStatusError, err.Error())
			} else {
				a.store.UpdateRuleStatus(rule.ID, models.RuleStatusRunning, "")
			}
		} else {
			// Sync status: if not enabled but status is running, reset to stopped
			if rule.Status == models.RuleStatusRunning {
				a.store.UpdateRuleStatus(rule.ID, models.RuleStatusStopped, "")
			}
		}
	}

	// Initialize tray and hotkey
	a.InitTray()
	a.InitHotkey()
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	// Stop tray and hotkey
	a.StopTrayAndHotkey()

	if a.useIPC {
		a.ipcClient.Close()
	} else if a.engine != nil {
		a.engine.StopAll()
	}
}

// ==================== Rule Operations ====================

// GetRules returns all rules
func (a *App) GetRules() []*models.Rule {
	if a.useIPC {
		rules, err := a.ipcClient.GetRules()
		if err != nil {
			log.Printf("[App] IPC GetRules error: %v", err)
			return []*models.Rule{}
		}
		return rules
	}
	return a.store.GetRules()
}

// GetRule returns a single rule by ID
func (a *App) GetRule(id string) *models.Rule {
	if a.useIPC {
		rule, err := a.ipcClient.GetRule(id)
		if err != nil {
			log.Printf("[App] IPC GetRule error: %v", err)
			return nil
		}
		return rule
	}
	rule, _ := a.store.GetRule(id)
	return rule
}

// CreateRule creates a new rule
func (a *App) CreateRule(rule *models.Rule) error {
	if a.useIPC {
		_, err := a.ipcClient.CreateRule(rule)
		return err
	}
	return a.store.CreateRule(rule)
}

// UpdateRule updates an existing rule
func (a *App) UpdateRule(rule *models.Rule) error {
	if a.useIPC {
		return a.ipcClient.UpdateRule(rule)
	}

	// Stop if running
	if a.engine.IsRunning(rule.ID) {
		a.engine.StopRule(rule.ID)
	}

	if err := a.store.UpdateRule(rule); err != nil {
		return err
	}

	// Restart if enabled
	if rule.Enabled {
		if err := a.engine.StartRule(rule); err != nil {
			a.store.UpdateRuleStatus(rule.ID, models.RuleStatusError, err.Error())
			return err
		}
		a.store.UpdateRuleStatus(rule.ID, models.RuleStatusRunning, "")
	}

	return nil
}

// DeleteRule deletes a rule
func (a *App) DeleteRule(id string) error {
	if a.useIPC {
		return a.ipcClient.DeleteRule(id)
	}

	// Stop if running
	if a.engine.IsRunning(id) {
		a.engine.StopRule(id)
	}

	return a.store.DeleteRule(id)
}

// StartRule starts a rule
func (a *App) StartRule(id string) error {
	log.Printf("[App] StartRule called for id: %s, useIPC: %v", id, a.useIPC)

	if a.useIPC {
		log.Printf("[App] StartRule: calling IPC client...")
		err := a.ipcClient.StartRule(id)
		if err != nil {
			log.Printf("[App] StartRule: IPC error: %v", err)
		} else {
			log.Printf("[App] StartRule: IPC call succeeded")
		}
		return err
	}

	log.Printf("[App] StartRule: running in embedded mode")
	rule, err := a.store.GetRule(id)
	if err != nil {
		return err
	}

	if err := a.engine.StartRule(rule); err != nil {
		a.store.UpdateRuleStatus(id, models.RuleStatusError, err.Error())
		return err
	}

	a.store.UpdateRuleStatus(id, models.RuleStatusRunning, "")
	return nil
}

// StopRule stops a rule
func (a *App) StopRule(id string) error {
	if a.useIPC {
		return a.ipcClient.StopRule(id)
	}

	// Try to stop the service (ignore "not running" error)
	if err := a.engine.StopRule(id); err != nil {
		// If service is not running, just update the status
		if err != models.ErrServiceNotRunning {
			return err
		}
	}

	// Always update status to stopped
	a.store.UpdateRuleStatus(id, models.RuleStatusStopped, "")
	return nil
}

// ==================== Chain Operations ====================

// GetChains returns all chains
func (a *App) GetChains() []*models.Chain {
	if a.useIPC {
		chains, err := a.ipcClient.GetChains()
		if err != nil {
			log.Printf("[App] IPC GetChains error: %v", err)
			return []*models.Chain{}
		}
		return chains
	}
	return a.store.GetChains()
}

// CreateChain creates a new chain
func (a *App) CreateChain(chain *models.Chain) error {
	if a.useIPC {
		_, err := a.ipcClient.CreateChain(chain)
		return err
	}

	if err := a.store.CreateChain(chain); err != nil {
		return err
	}
	a.engine.SetChains(a.store.GetChains())
	return nil
}

// UpdateChain updates an existing chain
func (a *App) UpdateChain(chain *models.Chain) error {
	if a.useIPC {
		return a.ipcClient.UpdateChain(chain)
	}

	if err := a.store.UpdateChain(chain); err != nil {
		return err
	}
	a.engine.SetChains(a.store.GetChains())
	return nil
}

// DeleteChain deletes a chain
func (a *App) DeleteChain(id string) error {
	if a.useIPC {
		return a.ipcClient.DeleteChain(id)
	}

	if err := a.store.DeleteChain(id); err != nil {
		return err
	}
	a.engine.SetChains(a.store.GetChains())
	return nil
}

// ==================== Config Operations ====================

// GetConfig returns the application configuration
func (a *App) GetConfig() *models.AppConfig {
	if a.useIPC {
		config, err := a.ipcClient.GetConfig()
		if err != nil {
			log.Printf("[App] IPC GetConfig error: %v", err)
			return models.DefaultAppConfig()
		}
		return config
	}
	return a.store.GetConfig()
}

// UpdateConfig updates the application configuration
func (a *App) UpdateConfig(config *models.AppConfig) error {
	if a.useIPC {
		return a.ipcClient.UpdateConfig(config)
	}
	return a.store.UpdateConfig(config)
}

// ==================== Status Operations ====================

// GetStatus returns the current application status
func (a *App) GetStatus() *models.ServiceStatus {
	if a.useIPC {
		status, err := a.ipcClient.GetStatus()
		if err != nil {
			return &models.ServiceStatus{
				Running: false,
				Version: "1.0.0",
			}
		}
		return status
	}

	rules := a.store.GetRules()
	runningIDs := a.engine.GetRunningRuleIDs()

	return &models.ServiceStatus{
		Running:     true,
		RulesActive: len(runningIDs),
		RulesTotal:  len(rules),
		Version:     "1.0.0",
	}
}

// IsServiceMode returns true if connected to background service
func (a *App) IsServiceMode() bool {
	return a.useIPC
}

// ==================== Service Management ====================

// InstallService installs the background service
func (a *App) InstallService() error {
	return daemon.Install()
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

// ExportData exports all data as JSON string
func (a *App) ExportData() string {
	if a.useIPC {
		data, err := a.ipcClient.ExportData()
		if err != nil {
			log.Printf("[App] IPC ExportData error: %v", err)
			return ""
		}
		return string(data)
	}

	data, err := a.store.ExportData()
	if err != nil {
		log.Printf("[App] ExportData error: %v", err)
		return ""
	}
	return string(data)
}

// ImportData imports data from JSON string
func (a *App) ImportData(data string, merge bool) error {
	if a.useIPC {
		return a.ipcClient.ImportData([]byte(data), merge)
	}

	var appData models.AppData
	if err := json.Unmarshal([]byte(data), &appData); err != nil {
		return err
	}

	if err := a.store.ImportData(&appData, merge); err != nil {
		return err
	}

	a.engine.SetChains(a.store.GetChains())
	return nil
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
	content, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// ClearAllData clears all rules and chains
func (a *App) ClearAllData() error {
	// Stop all running rules first
	for _, rule := range a.store.GetRules() {
		if rule.Status == models.RuleStatusRunning {
			a.engine.StopRule(rule.ID)
		}
	}

	// Clear data by importing empty data
	emptyData := &models.AppData{
		Config: a.store.GetConfig(),
		Rules:  []*models.Rule{},
		Chains: []*models.Chain{},
	}

	if err := a.store.ImportData(emptyData, false); err != nil {
		return err
	}

	a.engine.SetChains([]*models.Chain{})
	return nil
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
	if a.useIPC {
		// TODO: implement IPC call
		return &models.RuleStats{RuleID: ruleID}
	}
	if a.engine == nil {
		return &models.RuleStats{RuleID: ruleID}
	}
	return a.engine.GetRuleStats(ruleID)
}

// GetAllRuleStats returns statistics for all rules
func (a *App) GetAllRuleStats() map[string]*models.RuleStats {
	if a.useIPC {
		// TODO: implement IPC call
		return make(map[string]*models.RuleStats)
	}
	if a.engine == nil {
		return make(map[string]*models.RuleStats)
	}
	return a.engine.GetAllRuleStats()
}

// ==================== Log Operations ====================

// LogEntry represents a log entry for frontend
type LogEntry struct {
	ID        int64  `json:"id"`
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	RuleID    string `json:"ruleId,omitempty"`
	RuleName  string `json:"ruleName,omitempty"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
}

// GetLogs returns recent log entries
func (a *App) GetLogs(count int) []LogEntry {
	if a.useIPC {
		// TODO: implement IPC call
		return []LogEntry{}
	}
	if a.engine == nil {
		return []LogEntry{}
	}

	logs := a.engine.GetLogs(count)
	result := make([]LogEntry, len(logs))
	for i, l := range logs {
		result[i] = LogEntry{
			ID:        l.ID,
			Timestamp: l.Timestamp,
			Level:     string(l.Level),
			RuleID:    l.RuleID,
			RuleName:  l.RuleName,
			Message:   l.Message,
			Details:   l.Details,
		}
	}
	return result
}

// GetLogsSince returns log entries since a specific ID
func (a *App) GetLogsSince(sinceID int64) []LogEntry {
	if a.useIPC {
		// TODO: implement IPC call
		return []LogEntry{}
	}
	if a.engine == nil {
		return []LogEntry{}
	}

	logs := a.engine.GetLogsSince(sinceID)
	result := make([]LogEntry, len(logs))
	for i, l := range logs {
		result[i] = LogEntry{
			ID:        l.ID,
			Timestamp: l.Timestamp,
			Level:     string(l.Level),
			RuleID:    l.RuleID,
			RuleName:  l.RuleName,
			Message:   l.Message,
			Details:   l.Details,
		}
	}
	return result
}

// GetLogsByRule returns log entries for a specific rule
func (a *App) GetLogsByRule(ruleID string) []LogEntry {
	if a.useIPC {
		// TODO: implement IPC call
		return []LogEntry{}
	}
	if a.engine == nil {
		return []LogEntry{}
	}

	logs := a.engine.GetLogsByRule(ruleID)
	result := make([]LogEntry, len(logs))
	for i, l := range logs {
		result[i] = LogEntry{
			ID:        l.ID,
			Timestamp: l.Timestamp,
			Level:     string(l.Level),
			RuleID:    l.RuleID,
			RuleName:  l.RuleName,
			Message:   l.Message,
			Details:   l.Details,
		}
	}
	return result
}

// ClearLogs clears all log entries
func (a *App) ClearLogs() {
	if a.useIPC {
		// TODO: implement IPC call
		return
	}
	if a.engine != nil {
		a.engine.ClearLogs()
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

	log.Printf("[App] Hotkey updated: %s", hotkey.GetHotkeyString(config))
	return nil
}

// StopTrayAndHotkey stops tray and hotkey managers
func (a *App) StopTrayAndHotkey() {
	if a.trayManager != nil {
		a.trayManager.Stop()
	}
	if a.hotkeyMgr != nil {
		a.hotkeyMgr.Stop()
	}
}
