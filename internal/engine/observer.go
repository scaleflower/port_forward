package engine

import (
	"context"
	"log"
	"sync"

	"github.com/go-gost/core/observer"
	xstats "github.com/go-gost/x/observer/stats"
)

// StatsObserver implements the gost observer.Observer interface
// to collect traffic statistics from gost services
type StatsObserver struct {
	mu           sync.RWMutex
	serviceStats map[string]*ServiceStats
	onUpdate     func(serviceName string, stats *ServiceStats)
}

// ServiceStats holds aggregated statistics for a service
type ServiceStats struct {
	TotalConns   uint64
	CurrentConns uint64
	InputBytes   uint64
	OutputBytes  uint64
	TotalErrs    uint64
}

// NewStatsObserver creates a new statistics observer
func NewStatsObserver() *StatsObserver {
	return &StatsObserver{
		serviceStats: make(map[string]*ServiceStats),
	}
}

// SetOnUpdate sets a callback function that is called when stats are updated
func (o *StatsObserver) SetOnUpdate(fn func(serviceName string, stats *ServiceStats)) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.onUpdate = fn
}

// Observe implements the observer.Observer interface
func (o *StatsObserver) Observe(ctx context.Context, events []observer.Event, opts ...observer.Option) error {
	log.Printf("[StatsObserver] Received %d events", len(events))
	for _, event := range events {
		log.Printf("[StatsObserver] Event type: %T, EventType: %s", event, event.Type())
		switch e := event.(type) {
		case xstats.StatsEvent:
			log.Printf("[StatsObserver] StatsEvent: Service=%s, In=%d, Out=%d", e.Service, e.InputBytes, e.OutputBytes)
			o.handleStatsEvent(e)
		case *xstats.StatsEvent:
			log.Printf("[StatsObserver] *StatsEvent: Service=%s, In=%d, Out=%d", e.Service, e.InputBytes, e.OutputBytes)
			o.handleStatsEvent(*e)
		}
	}
	return nil
}

// handleStatsEvent processes a stats event from gost
func (o *StatsObserver) handleStatsEvent(e xstats.StatsEvent) {
	serviceName := e.Service
	if serviceName == "" {
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	// Get or create stats for this service
	stats, exists := o.serviceStats[serviceName]
	if !exists {
		stats = &ServiceStats{}
		o.serviceStats[serviceName] = stats
	}

	// Update stats - gost provides cumulative values when resetTraffic is false
	stats.TotalConns = e.TotalConns
	stats.CurrentConns = e.CurrentConns
	stats.InputBytes = e.InputBytes   // Direct assignment - already cumulative
	stats.OutputBytes = e.OutputBytes // Direct assignment - already cumulative
	stats.TotalErrs = e.TotalErrs

	// Call update callback if set
	if o.onUpdate != nil {
		statsCopy := &ServiceStats{
			TotalConns:   stats.TotalConns,
			CurrentConns: stats.CurrentConns,
			InputBytes:   stats.InputBytes,
			OutputBytes:  stats.OutputBytes,
			TotalErrs:    stats.TotalErrs,
		}
		go o.onUpdate(serviceName, statsCopy)
	}
}

// GetStats returns statistics for a specific service
func (o *StatsObserver) GetStats(serviceName string) *ServiceStats {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if stats, exists := o.serviceStats[serviceName]; exists {
		return &ServiceStats{
			TotalConns:   stats.TotalConns,
			CurrentConns: stats.CurrentConns,
			InputBytes:   stats.InputBytes,
			OutputBytes:  stats.OutputBytes,
			TotalErrs:    stats.TotalErrs,
		}
	}
	return &ServiceStats{}
}

// GetAllStats returns statistics for all services
func (o *StatsObserver) GetAllStats() map[string]*ServiceStats {
	o.mu.RLock()
	defer o.mu.RUnlock()

	result := make(map[string]*ServiceStats)
	for name, stats := range o.serviceStats {
		result[name] = &ServiceStats{
			TotalConns:   stats.TotalConns,
			CurrentConns: stats.CurrentConns,
			InputBytes:   stats.InputBytes,
			OutputBytes:  stats.OutputBytes,
			TotalErrs:    stats.TotalErrs,
		}
	}
	return result
}

// ResetStats resets statistics for a specific service
func (o *StatsObserver) ResetStats(serviceName string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if stats, exists := o.serviceStats[serviceName]; exists {
		stats.TotalConns = 0
		stats.CurrentConns = 0
		stats.InputBytes = 0
		stats.OutputBytes = 0
		stats.TotalErrs = 0
	}
}

// ResetAllStats resets all statistics
func (o *StatsObserver) ResetAllStats() {
	o.mu.Lock()
	defer o.mu.Unlock()

	for _, stats := range o.serviceStats {
		stats.TotalConns = 0
		stats.CurrentConns = 0
		stats.InputBytes = 0
		stats.OutputBytes = 0
		stats.TotalErrs = 0
	}
}

// RemoveStats removes statistics for a specific service
func (o *StatsObserver) RemoveStats(serviceName string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	delete(o.serviceStats, serviceName)
}
