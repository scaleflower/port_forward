package controller

import (
	"pfm/internal/models"
)

// ServiceController defines the interface for application operations
// regardless of whether running in local (embedded) or remote (service) mode.
type ServiceController interface {
	// Rule Operations
	GetRules() ([]*models.Rule, error)
	GetRule(id string) (*models.Rule, error)
	CreateRule(rule *models.Rule) error
	UpdateRule(rule *models.Rule) error
	DeleteRule(id string) error
	StartRule(id string) error
	StopRule(id string) error
	StartAllRules() error
	StopAllRules() error

	// Chain Operations
	GetChains() ([]*models.Chain, error)
	CreateChain(chain *models.Chain) error
	UpdateChain(chain *models.Chain) error
	DeleteChain(id string) error

	// Config Operations
	GetConfig() (*models.AppConfig, error)
	UpdateConfig(config *models.AppConfig) error

	// Status Operations
	GetStatus() (*models.ServiceStatus, error)

	// Stats Operations
	GetRuleStats(ruleID string) *models.RuleStats
	GetAllRuleStats() map[string]*models.RuleStats

	// Log Operations
	GetLogs(count int) ([]models.LogEntry, error)
	GetLogsSince(sinceID int64) ([]models.LogEntry, error)
	GetLogsByRule(ruleID string) ([]models.LogEntry, error)
	ClearLogs() error

	// Data Operations
	ImportData(data []byte, merge bool) error
	ExportData() ([]byte, error)
	ClearAllData() error
}
