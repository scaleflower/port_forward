package ipc

import (
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sync"
	"time"

	"pfm/internal/models"
)

// Client handles IPC communications with the background service
type Client struct {
	mu     sync.Mutex
	client *rpc.Client
}

// NewClient creates a new IPC client
func NewClient() *Client {
	return &Client{}
}

// Connect establishes a connection to the IPC server
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		return nil
	}

	socketPath := GetSocketPath()
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		return err
	}

	c.client = jsonrpc.NewClient(conn)
	return nil
}

// Close closes the connection
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return nil
}

// IsConnected returns true if connected to the server
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.client != nil
}

// ensureConnected ensures the client is connected
func (c *Client) ensureConnected() error {
	if c.client == nil {
		return c.Connect()
	}
	return nil
}

// call makes an RPC call with automatic reconnection
func (c *Client) call(method string, args interface{}, reply interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client == nil {
		socketPath := GetSocketPath()
		conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
		if err != nil {
			return err
		}
		c.client = jsonrpc.NewClient(conn)
	}

	err := c.client.Call("RPCHandler."+method, args, reply)
	if err != nil {
		// Connection might be broken, reset
		c.client.Close()
		c.client = nil
	}
	return err
}

// ==================== Rule Operations ====================

// GetRules returns all rules
func (c *Client) GetRules() ([]*models.Rule, error) {
	var rules []*models.Rule
	err := c.call("GetRules", &Empty{}, &rules)
	return rules, err
}

// GetRule returns a single rule by ID
func (c *Client) GetRule(id string) (*models.Rule, error) {
	var rule models.Rule
	err := c.call("GetRule", &id, &rule)
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// CreateRule creates a new rule
func (c *Client) CreateRule(rule *models.Rule) (string, error) {
	var id string
	err := c.call("CreateRule", &CreateRuleArgs{Rule: rule}, &id)
	return id, err
}

// UpdateRule updates an existing rule
func (c *Client) UpdateRule(rule *models.Rule) error {
	var success bool
	return c.call("UpdateRule", rule, &success)
}

// DeleteRule deletes a rule
func (c *Client) DeleteRule(id string) error {
	var success bool
	return c.call("DeleteRule", &id, &success)
}

// StartRule starts a rule
func (c *Client) StartRule(id string) error {
	var success bool
	return c.call("StartRule", &id, &success)
}

// StopRule stops a rule
func (c *Client) StopRule(id string) error {
	var success bool
	return c.call("StopRule", &id, &success)
}

// ==================== Chain Operations ====================

// GetChains returns all chains
func (c *Client) GetChains() ([]*models.Chain, error) {
	var chains []*models.Chain
	err := c.call("GetChains", &Empty{}, &chains)
	return chains, err
}

// CreateChain creates a new chain
func (c *Client) CreateChain(chain *models.Chain) (string, error) {
	var id string
	err := c.call("CreateChain", &CreateChainArgs{Chain: chain}, &id)
	return id, err
}

// UpdateChain updates an existing chain
func (c *Client) UpdateChain(chain *models.Chain) error {
	var success bool
	return c.call("UpdateChain", chain, &success)
}

// DeleteChain deletes a chain
func (c *Client) DeleteChain(id string) error {
	var success bool
	return c.call("DeleteChain", &id, &success)
}

// ==================== Config Operations ====================

// GetConfig returns the application configuration
func (c *Client) GetConfig() (*models.AppConfig, error) {
	var config models.AppConfig
	err := c.call("GetConfig", &Empty{}, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// UpdateConfig updates the application configuration
func (c *Client) UpdateConfig(config *models.AppConfig) error {
	var success bool
	return c.call("UpdateConfig", config, &success)
}

// ==================== Status Operations ====================

// GetStatus returns the service status
func (c *Client) GetStatus() (*models.ServiceStatus, error) {
	var status models.ServiceStatus
	err := c.call("GetStatus", &Empty{}, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// Ping checks if the service is reachable
func (c *Client) Ping() bool {
	_, err := c.GetStatus()
	return err == nil
}

// ==================== Import/Export Operations ====================

// ExportData exports all data as JSON
func (c *Client) ExportData() ([]byte, error) {
	var data []byte
	err := c.call("ExportData", &Empty{}, &data)
	return data, err
}

// ImportData imports data from JSON
func (c *Client) ImportData(data []byte, merge bool) error {
	var success bool
	return c.call("ImportData", &ImportDataArgs{Data: data, Merge: merge}, &success)
}

// ==================== Log Operations ====================

// GetLogs returns recent log entries
func (c *Client) GetLogs(count int) ([]*models.LogEntry, error) {
	var logs []*models.LogEntry
	err := c.call("GetLogs", &GetLogsArgs{Count: count}, &logs)
	return logs, err
}

// GetLogsSince returns log entries since a specific ID
func (c *Client) GetLogsSince(sinceID int64) ([]*models.LogEntry, error) {
	var logs []*models.LogEntry
	err := c.call("GetLogsSince", &GetLogsSinceArgs{SinceID: sinceID}, &logs)
	return logs, err
}

// GetLogsByRule returns log entries for a specific rule
func (c *Client) GetLogsByRule(ruleID string) ([]*models.LogEntry, error) {
	var logs []*models.LogEntry
	err := c.call("GetLogsByRule", &GetLogsByRuleArgs{RuleID: ruleID}, &logs)
	return logs, err
}

// ClearLogs clears all log entries
func (c *Client) ClearLogs() error {
	var success bool
	return c.call("ClearLogs", &Empty{}, &success)
}

// GetLogsArgs holds arguments for GetLogs
type GetLogsArgs struct {
	Count int `json:"count"`
}

// GetLogsSinceArgs holds arguments for GetLogsSince
type GetLogsSinceArgs struct {
	SinceID int64 `json:"sinceId"`
}

// GetLogsByRuleArgs holds arguments for GetLogsByRule
type GetLogsByRuleArgs struct {
	RuleID string `json:"ruleId"`
}
