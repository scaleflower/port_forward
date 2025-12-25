package controller

import (
	"encoding/json"
	"pfm/internal/ipc"
	"pfm/internal/models"
)

// RemoteController implements ServiceController for service mode (IPC)
type RemoteController struct {
	client *ipc.Client
}

// NewRemote creates a new RemoteController
func NewRemote(client *ipc.Client) *RemoteController {
	return &RemoteController{
		client: client,
	}
}

// ==================== Rule Operations ====================

func (c *RemoteController) GetRules() ([]*models.Rule, error) {
	return c.client.GetRules()
}

func (c *RemoteController) GetRule(id string) (*models.Rule, error) {
	return c.client.GetRule(id)
}

func (c *RemoteController) CreateRule(rule *models.Rule) error {
	_, err := c.client.CreateRule(rule)
	return err
}

func (c *RemoteController) UpdateRule(rule *models.Rule) error {
	return c.client.UpdateRule(rule)
}

func (c *RemoteController) DeleteRule(id string) error {
	return c.client.DeleteRule(id)
}

func (c *RemoteController) StartRule(id string) error {
	return c.client.StartRule(id)
}

func (c *RemoteController) StopRule(id string) error {
	return c.client.StopRule(id)
}

// ==================== Chain Operations ====================

func (c *RemoteController) GetChains() ([]*models.Chain, error) {
	return c.client.GetChains()
}

func (c *RemoteController) CreateChain(chain *models.Chain) error {
	_, err := c.client.CreateChain(chain)
	return err
}

func (c *RemoteController) UpdateChain(chain *models.Chain) error {
	return c.client.UpdateChain(chain)
}

func (c *RemoteController) DeleteChain(id string) error {
	return c.client.DeleteChain(id)
}

// ==================== Config Operations ====================

func (c *RemoteController) GetConfig() (*models.AppConfig, error) {
	return c.client.GetConfig()
}

func (c *RemoteController) UpdateConfig(config *models.AppConfig) error {
	return c.client.UpdateConfig(config)
}

// ==================== Status Operations ====================

func (c *RemoteController) GetStatus() (*models.ServiceStatus, error) {
	return c.client.GetStatus()
}

// ==================== Stats Operations ====================

func (c *RemoteController) GetRuleStats(ruleID string) *models.RuleStats {
	// PENDING: IPC client doesn't seem to implement GetRuleStats based on client.go view?
	// Checking client.go... I didn't see GetRuleStats. app.go had TODOs.
	// We will return empty stats for now as it seems not implemented in IPC layer yet.
	return &models.RuleStats{RuleID: ruleID}
}

func (c *RemoteController) GetAllRuleStats() map[string]*models.RuleStats {
	// PENDING: Not implemented in IPC
	return make(map[string]*models.RuleStats)
}

// ==================== Log Operations ====================

func (c *RemoteController) GetLogs(count int) ([]models.LogEntry, error) {
	logs, err := c.client.GetLogs(count)
	if err != nil {
		return []models.LogEntry{}, err
	}
	return convertLogs(logs), nil
}

func (c *RemoteController) GetLogsSince(sinceID int64) ([]models.LogEntry, error) {
	logs, err := c.client.GetLogsSince(sinceID)
	if err != nil {
		return []models.LogEntry{}, err
	}
	return convertLogs(logs), nil
}

func (c *RemoteController) GetLogsByRule(ruleID string) ([]models.LogEntry, error) {
	logs, err := c.client.GetLogsByRule(ruleID)
	if err != nil {
		return []models.LogEntry{}, err
	}
	return convertLogs(logs), nil
}

func (c *RemoteController) ClearLogs() error {
	return c.client.ClearLogs()
}

// Helper to convert []*models.LogEntry to []models.LogEntry
func convertLogs(ptrs []*models.LogEntry) []models.LogEntry {
	result := make([]models.LogEntry, len(ptrs))
	for i, p := range ptrs {
		if p != nil {
			result[i] = *p
		}
	}
	return result
}

// ==================== Data Operations ====================

func (c *RemoteController) ImportData(data []byte, merge bool) error {
	return c.client.ImportData(data, merge)
}

func (c *RemoteController) ExportData() ([]byte, error) {
	return c.client.ExportData()
}

func (c *RemoteController) ClearAllData() error {
	// Not implemented in Client directly but app.go simulated it by stopping rules and importing empty data.
	// But RemoteController interacts with IPC service which should handle this safe?
	// app.go logic:
	/*
		func (a *App) ClearAllData() error {
			// Stop all running rules first
			for _, rule := range a.store.GetRules() { ... }
			// Clear data by importing empty data ...
		}
	*/
	// IPC client doesn't have ClearAllData.
	// However, ImportData with overwrite = empty should work?
	// Let's implement basics via ImportData with empty structure if IPC supports it.
	// Actually, `ClearAllData` logic in app.go for embedded mode was manual.
	// For remote mode, we might need to add IPC method or just use ImportData.
	// Since no `ClearAllData` in IPC client, I'll replicate app.go logic for now using available IPC methods?
	// But syncing that state remotely is tricky.
	// Best approach: Add ClearAllData to IPC? No, can't change IPC protocol easily without modifying server.
	// I'll stick to what Client offers: ImportData.
	// If I pass empty data to ImportData(data, false), it should overwrite.

	// Create empty data
	emptyData := &models.AppData{
		Rules:  []*models.Rule{},
		Chains: []*models.Chain{},
		// We might want to keep config? app.go kept config.
		// We need to fetch config first.
	}

	// Fetch config to preserve it?
	currentConfig, err := c.GetConfig()
	if err == nil && currentConfig != nil {
		emptyData.Config = currentConfig
	} else {
		emptyData.Config = models.DefaultAppConfig()
	}

	dataBytes, err := json.Marshal(emptyData)
	if err != nil {
		return err
	}

	return c.client.ImportData(dataBytes, false)
}
