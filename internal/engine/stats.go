package engine

import (
	"sync"
	"sync/atomic"
	"time"

	"pfm/internal/models"
)

// StatsTracker tracks traffic statistics for all rules
type StatsTracker struct {
	mu    sync.RWMutex
	stats map[string]*RuleStatsEntry
}

// RuleStatsEntry holds statistics for a single rule
type RuleStatsEntry struct {
	BytesIn      int64
	BytesOut     int64
	Connections  int64
	ActiveConns  int32
	Errors       int64
	LastActivity time.Time
}

// NewStatsTracker creates a new statistics tracker
func NewStatsTracker() *StatsTracker {
	return &StatsTracker{
		stats: make(map[string]*RuleStatsEntry),
	}
}

// InitRule initializes statistics for a rule
func (t *StatsTracker) InitRule(ruleID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats[ruleID] = &RuleStatsEntry{
		LastActivity: time.Now(),
	}
}

// RemoveRule removes statistics for a rule
func (t *StatsTracker) RemoveRule(ruleID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.stats, ruleID)
}

// AddBytesIn adds incoming bytes for a rule
func (t *StatsTracker) AddBytesIn(ruleID string, n int64) {
	t.mu.RLock()
	entry, exists := t.stats[ruleID]
	t.mu.RUnlock()

	if exists {
		atomic.AddInt64(&entry.BytesIn, n)
		entry.LastActivity = time.Now()
	}
}

// AddBytesOut adds outgoing bytes for a rule
func (t *StatsTracker) AddBytesOut(ruleID string, n int64) {
	t.mu.RLock()
	entry, exists := t.stats[ruleID]
	t.mu.RUnlock()

	if exists {
		atomic.AddInt64(&entry.BytesOut, n)
		entry.LastActivity = time.Now()
	}
}

// IncrementConnections increments the connection count for a rule
func (t *StatsTracker) IncrementConnections(ruleID string) {
	t.mu.RLock()
	entry, exists := t.stats[ruleID]
	t.mu.RUnlock()

	if exists {
		atomic.AddInt64(&entry.Connections, 1)
		atomic.AddInt32(&entry.ActiveConns, 1)
		entry.LastActivity = time.Now()
	}
}

// DecrementActiveConnections decrements the active connection count
func (t *StatsTracker) DecrementActiveConnections(ruleID string) {
	t.mu.RLock()
	entry, exists := t.stats[ruleID]
	t.mu.RUnlock()

	if exists {
		atomic.AddInt32(&entry.ActiveConns, -1)
	}
}

// IncrementErrors increments the error count for a rule
func (t *StatsTracker) IncrementErrors(ruleID string) {
	t.mu.RLock()
	entry, exists := t.stats[ruleID]
	t.mu.RUnlock()

	if exists {
		atomic.AddInt64(&entry.Errors, 1)
	}
}

// GetStats returns statistics for a specific rule
func (t *StatsTracker) GetStats(ruleID string) *models.RuleStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entry, exists := t.stats[ruleID]
	if !exists {
		return &models.RuleStats{RuleID: ruleID}
	}

	return &models.RuleStats{
		RuleID:       ruleID,
		BytesIn:      atomic.LoadInt64(&entry.BytesIn),
		BytesOut:     atomic.LoadInt64(&entry.BytesOut),
		Connections:  atomic.LoadInt64(&entry.Connections),
		ActiveConns:  int(atomic.LoadInt32(&entry.ActiveConns)),
		Errors:       atomic.LoadInt64(&entry.Errors),
		LastActivity: entry.LastActivity.Format(time.RFC3339),
	}
}

// GetAllStats returns statistics for all rules
func (t *StatsTracker) GetAllStats() map[string]*models.RuleStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*models.RuleStats)
	for ruleID, entry := range t.stats {
		result[ruleID] = &models.RuleStats{
			RuleID:       ruleID,
			BytesIn:      atomic.LoadInt64(&entry.BytesIn),
			BytesOut:     atomic.LoadInt64(&entry.BytesOut),
			Connections:  atomic.LoadInt64(&entry.Connections),
			ActiveConns:  int(atomic.LoadInt32(&entry.ActiveConns)),
			Errors:       atomic.LoadInt64(&entry.Errors),
			LastActivity: entry.LastActivity.Format(time.RFC3339),
		}
	}
	return result
}

// ResetStats resets statistics for a specific rule
func (t *StatsTracker) ResetStats(ruleID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if entry, exists := t.stats[ruleID]; exists {
		atomic.StoreInt64(&entry.BytesIn, 0)
		atomic.StoreInt64(&entry.BytesOut, 0)
		atomic.StoreInt64(&entry.Connections, 0)
		atomic.StoreInt64(&entry.Errors, 0)
	}
}

// ResetAllStats resets all statistics
func (t *StatsTracker) ResetAllStats() {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, entry := range t.stats {
		atomic.StoreInt64(&entry.BytesIn, 0)
		atomic.StoreInt64(&entry.BytesOut, 0)
		atomic.StoreInt64(&entry.Connections, 0)
		atomic.StoreInt64(&entry.Errors, 0)
	}
}

// UpdateFromObserver updates statistics from the StatsObserver
func (t *StatsTracker) UpdateFromObserver(ruleID string, stats *ServiceStats) {
	t.mu.Lock()
	defer t.mu.Unlock()

	entry, exists := t.stats[ruleID]
	if !exists {
		entry = &RuleStatsEntry{
			LastActivity: time.Now(),
		}
		t.stats[ruleID] = entry
	}

	// Store the values from observer (these are cumulative)
	atomic.StoreInt64(&entry.BytesIn, int64(stats.InputBytes))
	atomic.StoreInt64(&entry.BytesOut, int64(stats.OutputBytes))
	atomic.StoreInt64(&entry.Connections, int64(stats.TotalConns))
	atomic.StoreInt32(&entry.ActiveConns, int32(stats.CurrentConns))
	atomic.StoreInt64(&entry.Errors, int64(stats.TotalErrs))
	entry.LastActivity = time.Now()
}
