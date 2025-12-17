package engine

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/go-gost/core/service"
	"github.com/go-gost/x/config/loader"
	"github.com/go-gost/x/registry"
	"pfm/internal/models"
)

// Engine manages gost services for port forwarding
type Engine struct {
	mu       sync.RWMutex
	services map[string]*serviceEntry
	chains   []*models.Chain
	logger   *log.Logger
}

// serviceEntry holds a running service and its metadata
type serviceEntry struct {
	service service.Service
	rule    *models.Rule
	cancel  context.CancelFunc
}

// New creates a new Engine instance
func New() *Engine {
	return &Engine{
		services: make(map[string]*serviceEntry),
		chains:   []*models.Chain{},
		logger:   log.Default(),
	}
}

// SetLogger sets the logger for the engine
func (e *Engine) SetLogger(logger *log.Logger) {
	e.logger = logger
}

// SetChains updates the chain configurations
func (e *Engine) SetChains(chains []*models.Chain) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.chains = chains
}

// StartRule starts a forwarding rule
func (e *Engine) StartRule(rule *models.Rule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if already running
	if _, exists := e.services[rule.ID]; exists {
		return models.ErrServiceRunning
	}

	// Validate rule
	if err := rule.Validate(); err != nil {
		return err
	}

	// Convert rule to gost config
	cfg, err := RuleToGostConfig(rule, e.chains)
	if err != nil {
		return &models.EngineError{
			RuleID:  rule.ID,
			Op:      "config",
			Message: "failed to build configuration",
			Err:     err,
		}
	}

	// Load the configuration
	if err := loader.Load(cfg); err != nil {
		return &models.EngineError{
			RuleID:  rule.ID,
			Op:      "load",
			Message: "failed to load configuration",
			Err:     err,
		}
	}

	// Get the registered service
	svc := registry.ServiceRegistry().Get(rule.ID)
	if svc == nil {
		return &models.EngineError{
			RuleID:  rule.ID,
			Op:      "registry",
			Message: "service not found in registry",
		}
	}

	// Create context for the service
	ctx, cancel := context.WithCancel(context.Background())

	// Store the service entry
	e.services[rule.ID] = &serviceEntry{
		service: svc,
		rule:    rule.Clone(),
		cancel:  cancel,
	}

	// Start the service in a goroutine
	go func() {
		e.logger.Printf("[Engine] Starting service: %s (%s)", rule.Name, rule.ID)
		if err := svc.Serve(); err != nil {
			select {
			case <-ctx.Done():
				// Service was stopped intentionally
			default:
				e.logger.Printf("[Engine] Service error: %s - %v", rule.ID, err)
			}
		}
	}()

	return nil
}

// StopRule stops a running rule
func (e *Engine) StopRule(id string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	entry, exists := e.services[id]
	if !exists {
		return models.ErrServiceNotRunning
	}

	e.logger.Printf("[Engine] Stopping service: %s", id)

	// Cancel the context
	if entry.cancel != nil {
		entry.cancel()
	}

	// Close the service
	if entry.service != nil {
		entry.service.Close()
	}

	// Unregister from registry
	registry.ServiceRegistry().Unregister(id)

	// Remove from map
	delete(e.services, id)

	return nil
}

// RestartRule restarts a rule
func (e *Engine) RestartRule(rule *models.Rule) error {
	// Stop if running
	e.StopRule(rule.ID)

	// Start with new configuration
	return e.StartRule(rule)
}

// StopAll stops all running services
func (e *Engine) StopAll() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Printf("[Engine] Stopping all services...")

	for id, entry := range e.services {
		if entry.cancel != nil {
			entry.cancel()
		}
		if entry.service != nil {
			entry.service.Close()
		}
		registry.ServiceRegistry().Unregister(id)
	}

	e.services = make(map[string]*serviceEntry)
}

// GetRunningRuleIDs returns IDs of all running rules
func (e *Engine) GetRunningRuleIDs() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	ids := make([]string, 0, len(e.services))
	for id := range e.services {
		ids = append(ids, id)
	}
	return ids
}

// IsRunning checks if a rule is currently running
func (e *Engine) IsRunning(id string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, exists := e.services[id]
	return exists
}

// GetStatus returns the status of all running services
func (e *Engine) GetStatus() map[string]*models.ServiceStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := make(map[string]*models.ServiceStatus)
	for id, entry := range e.services {
		status[id] = &models.ServiceStatus{
			Running:     true,
			RulesActive: len(e.services),
		}
		if entry.rule != nil {
			status[id].Version = fmt.Sprintf("Rule: %s", entry.rule.Name)
		}
	}
	return status
}

// GetRunningCount returns the number of running services
func (e *Engine) GetRunningCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.services)
}
