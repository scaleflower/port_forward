package daemon

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
	"pfm/internal/engine"
	"pfm/internal/ipc"
	"pfm/internal/models"
	"pfm/internal/storage"
)

const (
	ServiceName        = "PortForwardManager"
	ServiceDisplayName = "Port Forward Manager"
	ServiceDescription = "Port forwarding and proxy management service"
)

// Daemon represents the background service
type Daemon struct {
	engine    *engine.Engine
	store     *storage.Store
	ipcServer *ipc.Server
	logger    *log.Logger
	service   service.Service
}

// program implements service.Interface
type program struct {
	daemon *Daemon
}

func (p *program) Start(s service.Service) error {
	go p.daemon.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	return p.daemon.stop()
}

// New creates a new Daemon instance
func New() (*Daemon, error) {
	// Initialize storage
	store, err := storage.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize engine
	eng := engine.New()

	// Initialize IPC server
	ipcServer := ipc.NewServer(eng, store)

	// Setup logger
	logFile := filepath.Join(store.GetDataDir(), "service.log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	logger := log.New(f, "", log.LstdFlags|log.Lshortfile)

	eng.SetLogger(logger)
	ipcServer.SetLogger(logger)

	return &Daemon{
		engine:    eng,
		store:     store,
		ipcServer: ipcServer,
		logger:    logger,
	}, nil
}

// run starts the daemon services
func (d *Daemon) run() {
	d.logger.Println("[Daemon] Starting...")

	// Start IPC server
	if err := d.ipcServer.Start(); err != nil {
		d.logger.Printf("[Daemon] Failed to start IPC server: %v", err)
		return
	}

	// Load chains into engine
	chains := d.store.GetChains()
	d.engine.SetChains(chains)

	// Start enabled rules
	rules := d.store.GetRules()
	for _, rule := range rules {
		if rule.Enabled {
			if err := d.engine.StartRule(rule); err != nil {
				d.logger.Printf("[Daemon] Failed to start rule %s: %v", rule.Name, err)
				d.store.UpdateRuleStatus(rule.ID, models.RuleStatusError, err.Error())
			} else {
				d.store.UpdateRuleStatus(rule.ID, models.RuleStatusRunning, "")
				d.logger.Printf("[Daemon] Started rule: %s", rule.Name)
			}
		}
	}

	d.logger.Println("[Daemon] Started successfully")
}

// stop stops all daemon services
func (d *Daemon) stop() error {
	d.logger.Println("[Daemon] Stopping...")

	// Stop all rules
	d.engine.StopAll()

	// Stop IPC server
	d.ipcServer.Stop()

	d.logger.Println("[Daemon] Stopped")
	return nil
}

// Run runs the daemon as a service or standalone
func (d *Daemon) Run() error {
	svcConfig := &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
	}

	prg := &program{daemon: d}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	d.service = s
	return s.Run()
}

// Install installs the daemon as a system service
func Install() error {
	svcConfig := &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
	}

	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	svcConfig.Executable = execPath
	svcConfig.Arguments = []string{"service", "run"}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	return s.Install()
}

// Uninstall removes the daemon from system services
func Uninstall() error {
	svcConfig := &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	return s.Uninstall()
}

// Start starts the installed service
func Start() error {
	svcConfig := &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	return s.Start()
}

// Stop stops the installed service
func Stop() error {
	svcConfig := &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	return s.Stop()
}

// Restart restarts the installed service
func Restart() error {
	svcConfig := &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return err
	}

	return s.Restart()
}

// Status returns the status of the service
func Status() (service.Status, error) {
	svcConfig := &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return service.StatusUnknown, err
	}

	return s.Status()
}

// IsInstalled checks if the service is installed
func IsInstalled() bool {
	status, err := Status()
	if err != nil {
		return false
	}
	return status != service.StatusUnknown
}

// IsRunning checks if the service is running
func IsRunning() bool {
	status, err := Status()
	if err != nil {
		return false
	}
	return status == service.StatusRunning
}
