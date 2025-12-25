package controller

import (
	"encoding/json"

	"pfm/internal/engine"
	"pfm/internal/models"
	"pfm/internal/storage"
)

// LocalController implements ServiceController for embedded mode
type LocalController struct {
	engine *engine.Engine
	store  *storage.Store
}

// NewLocal creates a new LocalController
func NewLocal(engine *engine.Engine, store *storage.Store) *LocalController {
	return &LocalController{
		engine: engine,
		store:  store,
	}
}

// Init initializes the engine with data from store
func (c *LocalController) Init() error {
	// Set chains
	c.engine.SetChains(c.store.GetChains())

	// Set status change callback to sync engine errors to store
	c.engine.SetStatusChangeCallback(func(ruleID string, status string, errorMsg string) {
		// Log? app.go logged it.
		// log.Printf("[Controller] Status change: rule=%s, status=%s, error=%s", ruleID, status, errorMsg)
		if status == "error" {
			c.store.UpdateRuleStatus(ruleID, models.RuleStatusError, errorMsg)
		} else if status == "stopped" {
			c.store.UpdateRuleStatus(ruleID, models.RuleStatusStopped, "")
		}
	})

	// Start enabled rules and sync status
	for _, rule := range c.store.GetRules() {
		if rule.Enabled {
			if err := c.engine.StartRule(rule); err != nil {
				// log.Printf("[Controller] Failed to start rule %s: %v", rule.Name, err)
				c.store.UpdateRuleStatus(rule.ID, models.RuleStatusError, err.Error())
			} else {
				c.store.UpdateRuleStatus(rule.ID, models.RuleStatusRunning, "")
			}
		} else {
			// Sync status: if not enabled but status is running, reset to stopped
			if rule.Status == models.RuleStatusRunning {
				c.store.UpdateRuleStatus(rule.ID, models.RuleStatusStopped, "")
			}
		}
	}
	return nil
}

// ==================== Rule Operations ====================

func (c *LocalController) GetRules() ([]*models.Rule, error) {
	return c.store.GetRules(), nil
}

func (c *LocalController) GetRule(id string) (*models.Rule, error) {
	return c.store.GetRule(id)
}

func (c *LocalController) CreateRule(rule *models.Rule) error {
	return c.store.CreateRule(rule)
}

func (c *LocalController) UpdateRule(rule *models.Rule) error {
	// Stop if running
	if c.engine.IsRunning(rule.ID) {
		c.engine.StopRule(rule.ID)
	}

	if err := c.store.UpdateRule(rule); err != nil {
		return err
	}

	// Restart if enabled
	if rule.Enabled {
		if err := c.engine.StartRule(rule); err != nil {
			c.store.UpdateRuleStatus(rule.ID, models.RuleStatusError, err.Error())
			return err
		}
		c.store.UpdateRuleStatus(rule.ID, models.RuleStatusRunning, "")
	}

	return nil
}

func (c *LocalController) DeleteRule(id string) error {
	// Stop if running
	if c.engine.IsRunning(id) {
		c.engine.StopRule(id)
	}

	return c.store.DeleteRule(id)
}

func (c *LocalController) StartRule(id string) error {
	rule, err := c.store.GetRule(id)
	if err != nil {
		return err
	}

	if err := c.engine.StartRule(rule); err != nil {
		c.store.UpdateRuleStatus(id, models.RuleStatusError, err.Error())
		return err
	}

	c.store.UpdateRuleStatus(id, models.RuleStatusRunning, "")
	return nil
}

func (c *LocalController) StopRule(id string) error {
	// Try to stop the service (ignore "not running" error)
	if err := c.engine.StopRule(id); err != nil {
		// If service is not running, just update the status
		if err != models.ErrServiceNotRunning {
			return err
		}
	}

	// Always update status to stopped
	c.store.UpdateRuleStatus(id, models.RuleStatusStopped, "")
	return nil
}

func (c *LocalController) StartAllRules() error {
	rules := c.store.GetRules()
	var lastErr error
	for _, rule := range rules {
		// Only start if enabled (or force enable?)
		// "Batch Start" usually implies starting all enabled rules, or enabling and starting selected.
		// For simplicity, let's assume it attempts to start all *enabled* rules,
		// OR maybe the user intention is "Start All" meaning "Enable and Start All".
		// But usually "Start All" just starts what can be started.
		// If we stick to "Start Enabled", it's same as Init().
		// If the user selects rules and clicks "Batch Start", that's frontend logic calling StartRule multiple times.
		// If there is a global "Start All", it might mean start everything.
		// Let's assume it starts all existing rules (enabling them).

		// Actually, let's follow standard behavior: Start all *enabled* rules that are not running?
		// Or if it's a "Start All" button, it usually enables them.
		// Let's implement it as: Enable and Start All.

		rule.Enabled = true
		c.store.UpdateRule(rule) // persist enabled state

		if err := c.engine.StartRule(rule); err != nil {
			c.store.UpdateRuleStatus(rule.ID, models.RuleStatusError, err.Error())
			lastErr = err
		} else {
			c.store.UpdateRuleStatus(rule.ID, models.RuleStatusRunning, "")
		}
	}
	return lastErr
}

func (c *LocalController) StopAllRules() error {
	rules := c.store.GetRules()
	var lastErr error
	for _, rule := range rules {
		if c.engine.IsRunning(rule.ID) {
			if err := c.engine.StopRule(rule.ID); err != nil {
				lastErr = err
			}
		}
		// Update status and disable? Or just stop?
		// Usually "Stop All" just stops them but keeps "Enabled" toggle if it was a persistent preference?
		// But for pfm, "Enabled" usually tracks running state intent.
		// Let's update status to Stopped.
		c.store.UpdateRuleStatus(rule.ID, models.RuleStatusStopped, "")
	}
	return lastErr
}

// ==================== Chain Operations ====================

func (c *LocalController) GetChains() ([]*models.Chain, error) {
	return c.store.GetChains(), nil
}

func (c *LocalController) CreateChain(chain *models.Chain) error {
	if err := c.store.CreateChain(chain); err != nil {
		return err
	}
	c.engine.SetChains(c.store.GetChains())
	return nil
}

func (c *LocalController) UpdateChain(chain *models.Chain) error {
	if err := c.store.UpdateChain(chain); err != nil {
		return err
	}
	c.engine.SetChains(c.store.GetChains())
	return nil
}

func (c *LocalController) DeleteChain(id string) error {
	if err := c.store.DeleteChain(id); err != nil {
		return err
	}
	c.engine.SetChains(c.store.GetChains())
	return nil
}

// ==================== Config Operations ====================

func (c *LocalController) GetConfig() (*models.AppConfig, error) {
	return c.store.GetConfig(), nil
}

func (c *LocalController) UpdateConfig(config *models.AppConfig) error {
	return c.store.UpdateConfig(config)
}

// ==================== Status Operations ====================

func (c *LocalController) GetStatus() (*models.ServiceStatus, error) {
	rules := c.store.GetRules()
	runningIDs := c.engine.GetRunningRuleIDs()

	return &models.ServiceStatus{
		Running:     true,
		RulesActive: len(runningIDs),
		RulesTotal:  len(rules),
		Version:     "V1.1.0",
	}, nil
}

// ==================== Stats Operations ====================

func (c *LocalController) GetRuleStats(ruleID string) *models.RuleStats {
	if c.engine == nil {
		return &models.RuleStats{RuleID: ruleID}
	}
	return c.engine.GetRuleStats(ruleID)
}

func (c *LocalController) GetAllRuleStats() map[string]*models.RuleStats {
	if c.engine == nil {
		return make(map[string]*models.RuleStats)
	}
	return c.engine.GetAllRuleStats()
}

// ==================== Log Operations ====================

func (c *LocalController) GetLogs(count int) ([]models.LogEntry, error) {
	if c.engine == nil {
		return []models.LogEntry{}, nil
	}
	// engine.GetLogs returns []engine.LogEntry (or models.LogEntry if updated).
	// We need to ensure engine uses models.LogEntry.
	// For now assuming engine will be updated to return []models.LogEntry
	return c.engine.GetLogs(count), nil
}

func (c *LocalController) GetLogsSince(sinceID int64) ([]models.LogEntry, error) {
	if c.engine == nil {
		return []models.LogEntry{}, nil
	}
	return c.engine.GetLogsSince(sinceID), nil
}

func (c *LocalController) GetLogsByRule(ruleID string) ([]models.LogEntry, error) {
	if c.engine == nil {
		return []models.LogEntry{}, nil
	}
	return c.engine.GetLogsByRule(ruleID), nil
}

func (c *LocalController) ClearLogs() error {
	if c.engine != nil {
		c.engine.ClearLogs()
	}
	return nil
}

// ==================== Data Operations ====================

func (c *LocalController) ImportData(data []byte, merge bool) error {
	var appData models.AppData
	if err := json.Unmarshal(data, &appData); err != nil {
		return err
	}

	// Stop all current rules if not merging (full overwrite)
	if !merge {
		currentRules := c.store.GetRules()
		for _, rule := range currentRules {
			if c.engine.IsRunning(rule.ID) {
				c.engine.StopRule(rule.ID)
			}
		}
	}

	if err := c.store.ImportData(&appData, merge); err != nil {
		return err
	}

	// Sync chains
	c.engine.SetChains(c.store.GetChains())

	// Start enabled rules from import
	importedRules := c.store.GetRules()
	for _, rule := range importedRules {
		if rule.Enabled {
			// If merging, rule might already be running.
			// But ImportData (store) might have updated it.
			// Safer to restart if running, or start if stopped.
			if c.engine.IsRunning(rule.ID) {
				c.engine.StopRule(rule.ID)
			}
			if err := c.engine.StartRule(rule); err != nil {
				c.store.UpdateRuleStatus(rule.ID, models.RuleStatusError, err.Error())
			} else {
				c.store.UpdateRuleStatus(rule.ID, models.RuleStatusRunning, "")
			}
		}
	}
	return nil
}

func (c *LocalController) ExportData() ([]byte, error) {
	// Always read directly from data file
	return storage.ReadDataFile()
}

func (c *LocalController) ClearAllData() error {
	// Stop all running rules first
	for _, rule := range c.store.GetRules() {
		if rule.Status == models.RuleStatusRunning {
			c.engine.StopRule(rule.ID)
		}
	}

	// Clear data by importing empty data
	emptyData := &models.AppData{
		Config: c.store.GetConfig(),
		Rules:  []*models.Rule{},
		Chains: []*models.Chain{},
	}

	if err := c.store.ImportData(emptyData, false); err != nil {
		return err
	}

	c.engine.SetChains([]*models.Chain{})
	return nil
}
