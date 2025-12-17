package main

import (
	"context"
	"encoding/json"
	"log"
	"runtime"

	"pfm/internal/daemon"
	"pfm/internal/engine"
	"pfm/internal/ipc"
	"pfm/internal/models"
	"pfm/internal/storage"
)

// App struct represents the application
type App struct {
	ctx       context.Context
	engine    *engine.Engine
	store     *storage.Store
	ipcClient *ipc.Client
	useIPC    bool // true if connecting to background service
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
	if a.ipcClient.Ping() {
		a.useIPC = true
		log.Println("[App] Connected to background service")
		return
	}

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

	// Start enabled rules
	for _, rule := range a.store.GetRules() {
		if rule.Enabled {
			if err := a.engine.StartRule(rule); err != nil {
				log.Printf("[App] Failed to start rule %s: %v", rule.Name, err)
				a.store.UpdateRuleStatus(rule.ID, models.RuleStatusError, err.Error())
			} else {
				a.store.UpdateRuleStatus(rule.ID, models.RuleStatusRunning, "")
			}
		}
	}
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
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
	if a.useIPC {
		return a.ipcClient.StartRule(id)
	}

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

	if err := a.engine.StopRule(id); err != nil {
		return err
	}

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
