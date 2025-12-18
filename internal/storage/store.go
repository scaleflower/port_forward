package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"pfm/internal/models"
)

// Store manages persistent storage for application data
type Store struct {
	mu       sync.RWMutex
	dataDir  string
	dataFile string
	data     *models.AppData
}

// New creates a new Store instance
func New() (*Store, error) {
	var dataDir string

	// Check environment variable first (for Docker/custom deployments)
	if envDir := os.Getenv("PFM_DATA_DIR"); envDir != "" {
		dataDir = envDir
	} else {
		dataDir = getDefaultDataDir()
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	store := &Store{
		dataDir:  dataDir,
		dataFile: filepath.Join(dataDir, "data.json"),
		data:     models.NewAppData(),
	}

	// Load existing data if available
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

// NewWithPath creates a new Store with a custom data path
func NewWithPath(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	store := &Store{
		dataDir:  dataDir,
		dataFile: filepath.Join(dataDir, "data.json"),
		data:     models.NewAppData(),
	}

	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return store, nil
}

// load reads data from disk
func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.dataFile)
	if err != nil {
		return err
	}

	var appData models.AppData
	if err := json.Unmarshal(data, &appData); err != nil {
		return err
	}

	s.data = &appData
	return nil
}

// save writes data to disk
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.dataFile, data, 0644)
}

// GetDataDir returns the data directory path
func (s *Store) GetDataDir() string {
	return s.dataDir
}

// GetDataFile returns the data file path
func (s *Store) GetDataFile() string {
	return s.dataFile
}

// GetDefaultDataFile returns the default data file path (for direct file access)
func GetDefaultDataFile() string {
	var dataDir string
	if envDir := os.Getenv("PFM_DATA_DIR"); envDir != "" {
		dataDir = envDir
	} else {
		dataDir = getDefaultDataDir()
	}
	return filepath.Join(dataDir, "data.json")
}

// ReadDataFile reads the data file directly from disk
func ReadDataFile() ([]byte, error) {
	return os.ReadFile(GetDefaultDataFile())
}

// ==================== Config Operations ====================

// GetConfig returns the current application configuration
func (s *Store) GetConfig() *models.AppConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.Config
}

// UpdateConfig updates the application configuration
func (s *Store) UpdateConfig(config *models.AppConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data.Config = config
	return s.save()
}

// ==================== Rule Operations ====================

// GetRules returns all rules
func (s *Store) GetRules() []*models.Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*models.Rule, len(s.data.Rules))
	for i, r := range s.data.Rules {
		rules[i] = r.Clone()
	}
	return rules
}

// GetRule returns a rule by ID
func (s *Store) GetRule(id string) (*models.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.data.Rules {
		if r.ID == id {
			return r.Clone(), nil
		}
	}
	return nil, models.ErrRuleNotFound
}

// CreateRule adds a new rule
func (s *Store) CreateRule(rule *models.Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate ID
	for _, r := range s.data.Rules {
		if r.ID == rule.ID {
			return models.ErrRuleExists
		}
	}

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = rule.CreatedAt
	s.data.Rules = append(s.data.Rules, rule.Clone())
	return s.save()
}

// UpdateRule updates an existing rule
func (s *Store) UpdateRule(rule *models.Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.data.Rules {
		if r.ID == rule.ID {
			rule.UpdatedAt = time.Now()
			rule.CreatedAt = r.CreatedAt // Preserve creation time
			s.data.Rules[i] = rule.Clone()
			return s.save()
		}
	}
	return models.ErrRuleNotFound
}

// DeleteRule removes a rule by ID
func (s *Store) DeleteRule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, r := range s.data.Rules {
		if r.ID == id {
			s.data.Rules = append(s.data.Rules[:i], s.data.Rules[i+1:]...)
			return s.save()
		}
	}
	return models.ErrRuleNotFound
}

// UpdateRuleStatus updates the status of a rule
func (s *Store) UpdateRuleStatus(id string, status models.RuleStatus, errorMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, r := range s.data.Rules {
		if r.ID == id {
			r.Status = status
			r.ErrorMsg = errorMsg
			r.UpdatedAt = time.Now()
			return s.save()
		}
	}
	return models.ErrRuleNotFound
}

// ==================== Chain Operations ====================

// GetChains returns all chains
func (s *Store) GetChains() []*models.Chain {
	s.mu.RLock()
	defer s.mu.RUnlock()

	chains := make([]*models.Chain, len(s.data.Chains))
	for i, c := range s.data.Chains {
		chains[i] = c.Clone()
	}
	return chains
}

// GetChain returns a chain by ID
func (s *Store) GetChain(id string) (*models.Chain, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, c := range s.data.Chains {
		if c.ID == id {
			return c.Clone(), nil
		}
	}
	return nil, models.ErrChainNotFound
}

// CreateChain adds a new chain
func (s *Store) CreateChain(chain *models.Chain) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate ID
	for _, c := range s.data.Chains {
		if c.ID == chain.ID {
			return models.ErrChainExists
		}
	}

	chain.CreatedAt = time.Now()
	chain.UpdatedAt = chain.CreatedAt
	s.data.Chains = append(s.data.Chains, chain.Clone())
	return s.save()
}

// UpdateChain updates an existing chain
func (s *Store) UpdateChain(chain *models.Chain) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.data.Chains {
		if c.ID == chain.ID {
			chain.UpdatedAt = time.Now()
			chain.CreatedAt = c.CreatedAt
			s.data.Chains[i] = chain.Clone()
			return s.save()
		}
	}
	return models.ErrChainNotFound
}

// DeleteChain removes a chain by ID
func (s *Store) DeleteChain(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if chain is in use
	for _, r := range s.data.Rules {
		if r.ChainID == id {
			return models.ErrChainInUse
		}
	}

	for i, c := range s.data.Chains {
		if c.ID == id {
			s.data.Chains = append(s.data.Chains[:i], s.data.Chains[i+1:]...)
			return s.save()
		}
	}
	return models.ErrChainNotFound
}

// ==================== Bulk Operations ====================

// GetAllData returns all application data
func (s *Store) GetAllData() *models.AppData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := &models.AppData{
		Config: s.data.Config,
		Rules:  make([]*models.Rule, len(s.data.Rules)),
		Chains: make([]*models.Chain, len(s.data.Chains)),
	}
	for i, r := range s.data.Rules {
		data.Rules[i] = r.Clone()
	}
	for i, c := range s.data.Chains {
		data.Chains[i] = c.Clone()
	}
	return data
}

// ImportData imports data from another source
func (s *Store) ImportData(data *models.AppData, merge bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if merge {
		// Merge rules
		for _, newRule := range data.Rules {
			found := false
			for i, existingRule := range s.data.Rules {
				if existingRule.ID == newRule.ID {
					s.data.Rules[i] = newRule.Clone()
					found = true
					break
				}
			}
			if !found {
				s.data.Rules = append(s.data.Rules, newRule.Clone())
			}
		}
		// Merge chains
		for _, newChain := range data.Chains {
			found := false
			for i, existingChain := range s.data.Chains {
				if existingChain.ID == newChain.ID {
					s.data.Chains[i] = newChain.Clone()
					found = true
					break
				}
			}
			if !found {
				s.data.Chains = append(s.data.Chains, newChain.Clone())
			}
		}
	} else {
		// Replace all data
		s.data = data
	}

	return s.save()
}

// ExportData exports all data to JSON bytes
func (s *Store) ExportData() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return json.MarshalIndent(s.data, "", "  ")
}
