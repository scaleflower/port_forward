package engine

import (
	"fmt"
	"sync"
	"time"

	"pfm/internal/models"
)

// LogManager manages application logs with a circular buffer
type LogManager struct {
	mu       sync.RWMutex
	entries  []models.LogEntry
	maxSize  int
	nextID   int64
	onChange func(entry models.LogEntry)
}

// NewLogManager creates a new log manager
func NewLogManager(maxSize int) *LogManager {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &LogManager{
		entries: make([]models.LogEntry, 0, maxSize),
		maxSize: maxSize,
		nextID:  1,
	}
}

// SetOnChange sets a callback function that's called when a new log entry is added
func (m *LogManager) SetOnChange(fn func(entry models.LogEntry)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = fn
}

// Add adds a new log entry
func (m *LogManager) Add(level models.LogLevel, ruleID, ruleName, message string, details ...string) {
	m.mu.Lock()

	entry := models.LogEntry{
		ID:        m.nextID,
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level,
		RuleID:    ruleID,
		RuleName:  ruleName,
		Message:   message,
	}

	if len(details) > 0 {
		entry.Details = details[0]
	}

	m.nextID++

	// Circular buffer behavior
	if len(m.entries) >= m.maxSize {
		m.entries = append(m.entries[1:], entry)
	} else {
		m.entries = append(m.entries, entry)
	}

	callback := m.onChange
	m.mu.Unlock()

	// Call callback outside of lock
	if callback != nil {
		callback(entry)
	}
}

// Debug adds a debug level log entry
func (m *LogManager) Debug(ruleID, ruleName, message string, details ...string) {
	m.Add(models.LogLevelDebug, ruleID, ruleName, message, details...)
}

// Info adds an info level log entry
func (m *LogManager) Info(ruleID, ruleName, message string, details ...string) {
	m.Add(models.LogLevelInfo, ruleID, ruleName, message, details...)
}

// Warn adds a warning level log entry
func (m *LogManager) Warn(ruleID, ruleName, message string, details ...string) {
	m.Add(models.LogLevelWarn, ruleID, ruleName, message, details...)
}

// Error adds an error level log entry
func (m *LogManager) Error(ruleID, ruleName, message string, details ...string) {
	m.Add(models.LogLevelError, ruleID, ruleName, message, details...)
}

// LogConnection logs a connection event
func (m *LogManager) LogConnection(ruleID, ruleName, clientAddr, targetAddr string) {
	message := fmt.Sprintf("新连接: %s -> %s", clientAddr, targetAddr)
	m.Info(ruleID, ruleName, message)
}

// LogDisconnection logs a disconnection event
func (m *LogManager) LogDisconnection(ruleID, ruleName, clientAddr string, bytesIn, bytesOut int64) {
	message := fmt.Sprintf("连接断开: %s (接收: %s, 发送: %s)", clientAddr, formatBytes(bytesIn), formatBytes(bytesOut))
	m.Info(ruleID, ruleName, message)
}

// LogTransfer logs data transfer
func (m *LogManager) LogTransfer(ruleID, ruleName, direction string, bytes int64) {
	message := fmt.Sprintf("数据传输 [%s]: %s", direction, formatBytes(bytes))
	m.Debug(ruleID, ruleName, message)
}

// LogError logs an error
func (m *LogManager) LogError(ruleID, ruleName string, err error) {
	m.Error(ruleID, ruleName, "错误", err.Error())
}

// LogServiceStart logs service start
func (m *LogManager) LogServiceStart(ruleID, ruleName, listenAddr string) {
	message := fmt.Sprintf("服务启动: 监听 %s", listenAddr)
	m.Info(ruleID, ruleName, message)
}

// LogServiceStop logs service stop
func (m *LogManager) LogServiceStop(ruleID, ruleName string) {
	m.Info(ruleID, ruleName, "服务停止")
}

// GetAll returns all log entries
func (m *LogManager) GetAll() []models.LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]models.LogEntry, len(m.entries))
	copy(result, m.entries)
	return result
}

// GetRecent returns the most recent n entries
func (m *LogManager) GetRecent(n int) []models.LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if n <= 0 || n >= len(m.entries) {
		result := make([]models.LogEntry, len(m.entries))
		copy(result, m.entries)
		return result
	}

	start := len(m.entries) - n
	result := make([]models.LogEntry, n)
	copy(result, m.entries[start:])
	return result
}

// GetByRule returns log entries for a specific rule
func (m *LogManager) GetByRule(ruleID string) []models.LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []models.LogEntry
	for _, entry := range m.entries {
		if entry.RuleID == ruleID {
			result = append(result, entry)
		}
	}
	return result
}

// GetSince returns log entries since a specific ID
func (m *LogManager) GetSince(sinceID int64) []models.LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []models.LogEntry
	for _, entry := range m.entries {
		if entry.ID > sinceID {
			result = append(result, entry)
		}
	}
	return result
}

// Clear clears all log entries
func (m *LogManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make([]models.LogEntry, 0, m.maxSize)
}

// formatBytes formats bytes into human-readable string
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
