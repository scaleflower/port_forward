package engine

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"pfm/internal/models"

	"github.com/go-gost/core/observer/stats"
	"github.com/go-gost/core/service"
	"github.com/go-gost/x/registry"
	xservice "github.com/go-gost/x/service"
)

// Statuser is an interface for accessing gost service status
type Statuser interface {
	Status() *xservice.Status
}

const (
	// observerName is the name used to register our stats observer
	observerName = "pfm-stats-observer"
)

// StatusChangeCallback is called when a service status changes
type StatusChangeCallback func(ruleID string, status string, errorMsg string)

// Engine manages gost services for port forwarding
type Engine struct {
	mu             sync.RWMutex
	services       map[string]*serviceEntry
	chains         []*models.Chain
	logger         *log.Logger
	stats          *StatsTracker
	logMgr         *LogManager
	observer       *StatsObserver
	pollCtx        context.Context
	pollCancel     context.CancelFunc
	onStatusChange StatusChangeCallback
}

// serviceEntry holds a running service and its metadata
type serviceEntry struct {
	service service.Service
	rule    *models.Rule
	cancel  context.CancelFunc
}

// New creates a new Engine instance
func New() *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	e := &Engine{
		services:   make(map[string]*serviceEntry),
		chains:     []*models.Chain{},
		logger:     log.Default(),
		stats:      NewStatsTracker(),
		logMgr:     NewLogManager(1000),
		observer:   NewStatsObserver(),
		pollCtx:    ctx,
		pollCancel: cancel,
	}

	// Set up observer callback to sync with StatsTracker
	e.observer.SetOnUpdate(func(serviceName string, stats *ServiceStats) {
		e.stats.UpdateFromObserver(serviceName, stats)
	})

	// Register our observer with gost registry
	// Unregister first to handle restart/test scenarios
	registry.ObserverRegistry().Unregister(observerName)
	if err := registry.ObserverRegistry().Register(observerName, e.observer); err != nil {
		log.Printf("[Engine] Failed to register observer: %v", err)
	} else {
		log.Printf("[Engine] Observer '%s' registered successfully", observerName)
	}

	// Verify registration
	if obs := registry.ObserverRegistry().Get(observerName); obs != nil {
		log.Printf("[Engine] Observer '%s' verified in registry", observerName)
	} else {
		log.Printf("[Engine] WARNING: Observer '%s' NOT found in registry!", observerName)
	}

	// Start polling goroutine to collect stats from gost services
	go e.pollStats()

	return e
}

// SetLogger sets the logger for the engine
func (e *Engine) SetLogger(logger *log.Logger) {
	e.logger = logger
}

// SetStatusChangeCallback sets the callback for status changes
func (e *Engine) SetStatusChangeCallback(callback StatusChangeCallback) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onStatusChange = callback
}

// SetChains updates the chain configurations
func (e *Engine) SetChains(chains []*models.Chain) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.chains = chains
}

// GetStats returns the stats tracker
func (e *Engine) GetStats() *StatsTracker {
	return e.stats
}

// GetLogManager returns the log manager
func (e *Engine) GetLogManager() *LogManager {
	return e.logMgr
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

	// Build service using builder
	svc, err := BuildService(rule, e.chains)
	if err != nil {
		e.logMgr.Error(rule.ID, rule.Name, "启动失败", err.Error())
		return &models.EngineError{
			RuleID:  rule.ID,
			Op:      "build",
			Message: "failed to build service",
			Err:     err,
		}
	}

	// Create context for the service
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize stats for this rule
	e.stats.InitRule(rule.ID)

	// Store the service entry
	e.services[rule.ID] = &serviceEntry{
		service: svc,
		rule:    rule.Clone(),
		cancel:  cancel,
	}

	// Log service start
	e.logMgr.LogServiceStart(rule.ID, rule.Name, rule.GetListenAddr())

	// Start the service in a goroutine
	go func() {
		ruleID := rule.ID
		ruleName := rule.Name
		e.logger.Printf("[Engine] Starting service: %s (%s)", ruleName, ruleID)
		if err := svc.Serve(); err != nil {
			select {
			case <-ctx.Done():
				// Service was stopped intentionally
			default:
				errMsg := err.Error()
				// "use of closed network connection" is a normal shutdown signal, not an error
				// Target unreachable errors should not stop the service
				if strings.Contains(errMsg, "use of closed network connection") {
					e.logger.Printf("[Engine] Service closed normally: %s", ruleID)
					return
				}

				e.logger.Printf("[Engine] Service error: %s - %v", ruleID, err)
				e.logMgr.LogError(ruleID, ruleName, err)
				e.stats.IncrementErrors(ruleID)

				// Only mark as error and remove for critical listener errors
				// (e.g., port already in use, permission denied)
				if strings.Contains(errMsg, "address already in use") ||
					strings.Contains(errMsg, "permission denied") ||
					strings.Contains(errMsg, "bind:") {
					// Notify status change to error
					e.mu.RLock()
					callback := e.onStatusChange
					e.mu.RUnlock()
					if callback != nil {
						callback(ruleID, "error", errMsg)
					}

					// Remove from running services
					e.mu.Lock()
					delete(e.services, ruleID)
					e.mu.Unlock()
				}
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

	ruleName := ""
	if entry.rule != nil {
		ruleName = entry.rule.Name
	}

	e.logger.Printf("[Engine] Stopping service: %s", id)
	e.logMgr.LogServiceStop(id, ruleName)

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

	// Remove stats (keep them for history view)
	// e.stats.RemoveRule(id)

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
	// Stop the stats polling goroutine
	if e.pollCancel != nil {
		e.pollCancel()
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.logger.Printf("[Engine] Stopping all services...")
	e.logMgr.Info("", "", "停止所有服务")

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
			status[id].Version = entry.rule.Name
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

// GetRuleStats returns statistics for a specific rule
func (e *Engine) GetRuleStats(ruleID string) *models.RuleStats {
	return e.stats.GetStats(ruleID)
}

// GetAllRuleStats returns statistics for all rules
func (e *Engine) GetAllRuleStats() map[string]*models.RuleStats {
	return e.stats.GetAllStats()
}

// GetLogs returns recent log entries
func (e *Engine) GetLogs(count int) []models.LogEntry {
	return e.logMgr.GetRecent(count)
}

// GetLogsSince returns log entries since a specific ID
func (e *Engine) GetLogsSince(sinceID int64) []models.LogEntry {
	return e.logMgr.GetSince(sinceID)
}

// GetLogsByRule returns log entries for a specific rule
func (e *Engine) GetLogsByRule(ruleID string) []models.LogEntry {
	return e.logMgr.GetByRule(ruleID)
}

// ClearLogs clears all log entries
func (e *Engine) ClearLogs() {
	e.logMgr.Clear()
}

// AddTrafficIn adds incoming traffic stats for a rule
func (e *Engine) AddTrafficIn(ruleID string, bytes int64) {
	e.stats.AddBytesIn(ruleID, bytes)
}

// AddTrafficOut adds outgoing traffic stats for a rule
func (e *Engine) AddTrafficOut(ruleID string, bytes int64) {
	e.stats.AddBytesOut(ruleID, bytes)
}

// IncrementConnections increments the connection count for a rule
func (e *Engine) IncrementConnections(ruleID string) {
	e.stats.IncrementConnections(ruleID)
}

// DecrementActiveConnections decrements the active connection count
func (e *Engine) DecrementActiveConnections(ruleID string) {
	e.stats.DecrementActiveConnections(ruleID)
}

// pollStats periodically polls gost services for statistics
func (e *Engine) pollStats() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Printf("[Engine] Stats polling started (every 2s)")

	for {
		select {
		case <-e.pollCtx.Done():
			log.Printf("[Engine] Stats polling stopped")
			return
		case <-ticker.C:
			e.collectStats()
		}
	}
}

// collectStats collects statistics from all running gost services
func (e *Engine) collectStats() {
	e.mu.RLock()
	serviceIDs := make([]string, 0, len(e.services))
	for id := range e.services {
		serviceIDs = append(serviceIDs, id)
	}
	e.mu.RUnlock()

	for _, id := range serviceIDs {
		svc := registry.ServiceRegistry().Get(id)
		if svc == nil {
			continue
		}

		// Type assert to access Status() method
		if statuser, ok := svc.(Statuser); ok {
			status := statuser.Status()
			if status != nil {
				st := status.Stats()
				if st != nil {
					// Get stats values from gost Stats interface
					bytesIn := st.Get(stats.KindInputBytes)
					bytesOut := st.Get(stats.KindOutputBytes)
					totalConns := st.Get(stats.KindTotalConns)
					currentConns := st.Get(stats.KindCurrentConns)
					totalErrs := st.Get(stats.KindTotalErrs)

					// Update our stats tracker with cumulative values
					e.stats.UpdateFromObserver(id, &ServiceStats{
						InputBytes:   bytesIn,
						OutputBytes:  bytesOut,
						TotalConns:   totalConns,
						CurrentConns: currentConns,
						TotalErrs:    totalErrs,
					})

					// Debug log for first few updates to verify it's working
					if bytesIn > 0 || bytesOut > 0 {
						log.Printf("[Engine] Stats for %s: In=%d, Out=%d, Conns=%d/%d, Errs=%d",
							id, bytesIn, bytesOut, currentConns, totalConns, totalErrs)
					}
				}
			}
		}
	}
}
