package ipc

import (
	"encoding/json"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"pfm/internal/engine"
	"pfm/internal/models"
	"pfm/internal/storage"
)

// Server handles IPC communications from GUI clients
type Server struct {
	mu       sync.Mutex
	engine   *engine.Engine
	store    *storage.Store
	listener net.Listener
	handler  *RPCHandler
	logger   *log.Logger
	running  bool
}

// NewServer creates a new IPC server
func NewServer(e *engine.Engine, s *storage.Store) *Server {
	return &Server{
		engine: e,
		store:  s,
		logger: log.Default(),
	}
}

// SetLogger sets the logger for the server
func (s *Server) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// Start starts the IPC server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	socketPath := GetSocketPath()

	// Ensure directory exists
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Remove existing socket file
	os.Remove(socketPath)

	var err error
	s.listener, err = net.Listen("unix", socketPath)
	if err != nil {
		return err
	}

	// Set permissions on socket
	os.Chmod(socketPath, 0600)

	// Create and register RPC handler
	s.handler = &RPCHandler{
		engine: s.engine,
		store:  s.store,
		logger: s.logger,
	}
	rpc.Register(s.handler)

	s.running = true
	s.logger.Printf("[IPC Server] Listening on %s", socketPath)

	// Accept connections
	go s.acceptLoop()

	return nil
}

// acceptLoop handles incoming connections
func (s *Server) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			s.mu.Lock()
			running := s.running
			s.mu.Unlock()
			if !running {
				return
			}
			s.logger.Printf("[IPC Server] Accept error: %v", err)
			continue
		}
		go jsonrpc.ServeConn(conn)
	}
}

// Stop stops the IPC server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	if s.listener != nil {
		s.listener.Close()
	}

	// Clean up socket file
	os.Remove(GetSocketPath())

	s.logger.Printf("[IPC Server] Stopped")
	return nil
}

// GetSocketPath returns the platform-specific socket path
func GetSocketPath() string {
	if runtime.GOOS == "windows" {
		return `\\.\pipe\pfm`
	}
	// Unix systems
	return filepath.Join(os.TempDir(), "pfm.sock")
}

// RPCHandler handles RPC method calls
type RPCHandler struct {
	engine *engine.Engine
	store  *storage.Store
	logger *log.Logger
}

// Empty is used for RPC methods with no arguments
type Empty struct{}

// ==================== Rule Operations ====================

// GetRules returns all rules
func (h *RPCHandler) GetRules(args *Empty, reply *[]*models.Rule) error {
	*reply = h.store.GetRules()
	return nil
}

// GetRule returns a single rule by ID
func (h *RPCHandler) GetRule(id *string, reply *models.Rule) error {
	rule, err := h.store.GetRule(*id)
	if err != nil {
		return err
	}
	*reply = *rule
	return nil
}

// CreateRuleArgs holds arguments for CreateRule
type CreateRuleArgs struct {
	Rule *models.Rule `json:"rule"`
}

// CreateRule creates a new rule
func (h *RPCHandler) CreateRule(args *CreateRuleArgs, reply *string) error {
	if err := h.store.CreateRule(args.Rule); err != nil {
		return err
	}
	*reply = args.Rule.ID
	return nil
}

// UpdateRule updates an existing rule
func (h *RPCHandler) UpdateRule(rule *models.Rule, reply *bool) error {
	// Stop if running
	if h.engine.IsRunning(rule.ID) {
		h.engine.StopRule(rule.ID)
	}

	if err := h.store.UpdateRule(rule); err != nil {
		*reply = false
		return err
	}

	// Restart if was enabled
	if rule.Enabled {
		if err := h.engine.StartRule(rule); err != nil {
			h.store.UpdateRuleStatus(rule.ID, models.RuleStatusError, err.Error())
			*reply = false
			return err
		}
		h.store.UpdateRuleStatus(rule.ID, models.RuleStatusRunning, "")
	}

	*reply = true
	return nil
}

// DeleteRule deletes a rule
func (h *RPCHandler) DeleteRule(id *string, reply *bool) error {
	// Stop if running
	if h.engine.IsRunning(*id) {
		h.engine.StopRule(*id)
	}

	if err := h.store.DeleteRule(*id); err != nil {
		*reply = false
		return err
	}
	*reply = true
	return nil
}

// StartRule starts a rule
func (h *RPCHandler) StartRule(id *string, reply *bool) error {
	rule, err := h.store.GetRule(*id)
	if err != nil {
		*reply = false
		return err
	}

	if err := h.engine.StartRule(rule); err != nil {
		h.store.UpdateRuleStatus(*id, models.RuleStatusError, err.Error())
		*reply = false
		return err
	}

	h.store.UpdateRuleStatus(*id, models.RuleStatusRunning, "")
	*reply = true
	return nil
}

// StopRule stops a rule
func (h *RPCHandler) StopRule(id *string, reply *bool) error {
	if err := h.engine.StopRule(*id); err != nil {
		*reply = false
		return err
	}

	h.store.UpdateRuleStatus(*id, models.RuleStatusStopped, "")
	*reply = true
	return nil
}

// ==================== Chain Operations ====================

// GetChains returns all chains
func (h *RPCHandler) GetChains(args *Empty, reply *[]*models.Chain) error {
	*reply = h.store.GetChains()
	return nil
}

// CreateChainArgs holds arguments for CreateChain
type CreateChainArgs struct {
	Chain *models.Chain `json:"chain"`
}

// CreateChain creates a new chain
func (h *RPCHandler) CreateChain(args *CreateChainArgs, reply *string) error {
	if err := h.store.CreateChain(args.Chain); err != nil {
		return err
	}
	// Update engine's chain list
	h.engine.SetChains(h.store.GetChains())
	*reply = args.Chain.ID
	return nil
}

// UpdateChain updates an existing chain
func (h *RPCHandler) UpdateChain(chain *models.Chain, reply *bool) error {
	if err := h.store.UpdateChain(chain); err != nil {
		*reply = false
		return err
	}
	// Update engine's chain list
	h.engine.SetChains(h.store.GetChains())
	*reply = true
	return nil
}

// DeleteChain deletes a chain
func (h *RPCHandler) DeleteChain(id *string, reply *bool) error {
	if err := h.store.DeleteChain(*id); err != nil {
		*reply = false
		return err
	}
	// Update engine's chain list
	h.engine.SetChains(h.store.GetChains())
	*reply = true
	return nil
}

// ==================== Config Operations ====================

// GetConfig returns the application configuration
func (h *RPCHandler) GetConfig(args *Empty, reply *models.AppConfig) error {
	*reply = *h.store.GetConfig()
	return nil
}

// UpdateConfig updates the application configuration
func (h *RPCHandler) UpdateConfig(config *models.AppConfig, reply *bool) error {
	if err := h.store.UpdateConfig(config); err != nil {
		*reply = false
		return err
	}
	*reply = true
	return nil
}

// ==================== Status Operations ====================

// GetStatus returns the service status
func (h *RPCHandler) GetStatus(args *Empty, reply *models.ServiceStatus) error {
	rules := h.store.GetRules()
	runningIDs := h.engine.GetRunningRuleIDs()

	*reply = models.ServiceStatus{
		Running:     true,
		RulesActive: len(runningIDs),
		RulesTotal:  len(rules),
		Version:     "1.0.0",
	}
	return nil
}

// ==================== Import/Export Operations ====================

// ExportData exports all data as JSON
func (h *RPCHandler) ExportData(args *Empty, reply *[]byte) error {
	data, err := h.store.ExportData()
	if err != nil {
		return err
	}
	*reply = data
	return nil
}

// ImportDataArgs holds arguments for ImportData
type ImportDataArgs struct {
	Data  []byte `json:"data"`
	Merge bool   `json:"merge"`
}

// ImportData imports data from JSON
func (h *RPCHandler) ImportData(args *ImportDataArgs, reply *bool) error {
	var data models.AppData
	if err := json.Unmarshal(args.Data, &data); err != nil {
		*reply = false
		return err
	}
	if err := h.store.ImportData(&data, args.Merge); err != nil {
		*reply = false
		return err
	}
	// Update engine's chain list
	h.engine.SetChains(h.store.GetChains())
	*reply = true
	return nil
}
