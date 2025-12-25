package daemon

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"pfm/internal/engine"
	"pfm/internal/ipc"
	"pfm/internal/models"
	"pfm/internal/storage"

	"github.com/kardianos/service"
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

	// Start IPC server - this is important but not fatal
	// If it fails, the GUI won't be able to communicate with the service,
	// but the port forwarding rules can still work
	if err := d.ipcServer.Start(); err != nil {
		d.logger.Printf("[Daemon] Warning: Failed to start IPC server: %v", err)
		d.logger.Println("[Daemon] Service will continue in degraded mode (no GUI communication)")
		// Continue running - don't return
	}

	// Load chains into engine
	chains := d.store.GetChains()
	d.engine.SetChains(chains)

	// Start enabled rules
	rules := d.store.GetRules()
	startedCount := 0
	failedCount := 0
	for _, rule := range rules {
		if rule.Enabled {
			if err := d.engine.StartRule(rule); err != nil {
				d.logger.Printf("[Daemon] Failed to start rule %s: %v", rule.Name, err)
				d.store.UpdateRuleStatus(rule.ID, models.RuleStatusError, err.Error())
				failedCount++
			} else {
				d.store.UpdateRuleStatus(rule.ID, models.RuleStatusRunning, "")
				d.logger.Printf("[Daemon] Started rule: %s", rule.Name)
				startedCount++
			}
		}
	}

	d.logger.Printf("[Daemon] Started successfully (%d rules started, %d failed)", startedCount, failedCount)
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
	// Check for admin privileges on Windows
	if runtime.GOOS == "windows" && !isAdmin() {
		return fmt.Errorf("需要管理员权限。请右键点击程序，选择「以管理员身份运行」，或在管理员命令提示符中运行：pfm.exe service install")
	}

	// Get executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve any symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	log.Printf("[Daemon] Installing service with executable: %s", execPath)

	// Check if service needs reinstall (installed but pointing to different path)
	if NeedsReinstall() {
		log.Printf("[Daemon] Service installed with different path, will reinstall...")
		oldPath := GetInstalledServicePath()
		log.Printf("[Daemon] Old path: %s, New path: %s", oldPath, execPath)

		// Stop and uninstall the old service first
		if IsRunning() {
			log.Printf("[Daemon] Stopping old service...")
			if err := Stop(); err != nil {
				log.Printf("[Daemon] Warning: failed to stop old service: %v", err)
			}
		}

		log.Printf("[Daemon] Uninstalling old service...")
		if err := Uninstall(); err != nil {
			log.Printf("[Daemon] Warning: failed to uninstall old service: %v", err)
			// Continue anyway, the install might still work
		}
	}

	// On macOS, use launchd plist directly with admin privileges
	if runtime.GOOS == "darwin" {
		return installDarwinService(execPath)
	}

	// For other platforms, use kardianos/service
	svcConfig := &service.Config{
		Name:        ServiceName,
		DisplayName: ServiceDisplayName,
		Description: ServiceDescription,
		Executable:  execPath,
		Arguments:   []string{"service", "run"},
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Check current status
	status, err := s.Status()
	if err == nil && status == service.StatusRunning {
		return fmt.Errorf("服务已在运行中，请先停止服务")
	}

	// If service exists (but not running), allow re-installation
	// This handles cases where NeedsReinstall failed to detect path mismatch (e.g. localized sc output)
	// or user just wants to repair/reinstall.
	if err == nil && status != service.StatusUnknown {
		log.Printf("[Daemon] Service exists (Status: %v), uninstalling before install...", status)
		if err := s.Uninstall(); err != nil {
			log.Printf("[Daemon] Warning: failed to uninstall existing service: %v", err)
		} else {
			// Wait a bit for Windows to release the service handle
			// time.Sleep(1 * time.Second) // "time" package needed
		}
	}

	if err := s.Install(); err != nil {
		if runtime.GOOS == "windows" {
			return fmt.Errorf("安装服务失败: %w\n\n请确保：\n1. 以管理员身份运行程序\n2. 将程序复制到固定目录（如 C:\\Tools\\pfm）\n3. 程序路径: %s", err, execPath)
		}
		return fmt.Errorf("failed to install service: %w (executable: %s)", err, execPath)
	}

	log.Printf("[Daemon] Service installed successfully")
	return nil
}

// installDarwinService installs service on macOS with admin privileges dialog
func installDarwinService(execPath string) error {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", ServiceName)

	// Get current user's home directory for config files
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/var/root"
	}

	// Create plist content with environment variables
	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>service</string>
        <string>run</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>HOME</key>
        <string>%s</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/%s.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/%s.err</string>
</dict>
</plist>`, ServiceName, execPath, homeDir, ServiceName, ServiceName)

	// Create temp file with plist content
	tmpFile, err := os.CreateTemp("", "pfm-*.plist")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(plistContent); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}
	tmpFile.Close()

	// Use AppleScript to run commands with admin privileges
	// This will show macOS authorization dialog
	script := fmt.Sprintf(`
do shell script "cp '%s' '%s' && launchctl load -w '%s'" with administrator privileges
`, tmpFile.Name(), plistPath, plistPath)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if strings.Contains(outputStr, "User canceled") || strings.Contains(outputStr, "-128") {
			return fmt.Errorf("用户取消了授权")
		}
		return fmt.Errorf("安装服务失败: %s", outputStr)
	}

	log.Printf("[Daemon] macOS service installed successfully")
	return nil
}

// Uninstall removes the daemon from system services
func Uninstall() error {
	// On macOS, use launchctl directly with admin privileges
	if runtime.GOOS == "darwin" {
		return uninstallDarwinService()
	}

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

// uninstallDarwinService uninstalls service on macOS with admin privileges
func uninstallDarwinService() error {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", ServiceName)

	// Check if plist exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return fmt.Errorf("服务未安装")
	}

	// Use AppleScript to run commands with admin privileges
	script := fmt.Sprintf(`
do shell script "launchctl unload -w '%s' 2>/dev/null; rm -f '%s'" with administrator privileges
`, plistPath, plistPath)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if strings.Contains(outputStr, "User canceled") || strings.Contains(outputStr, "-128") {
			return fmt.Errorf("用户取消了授权")
		}
		return fmt.Errorf("卸载服务失败: %s", outputStr)
	}

	log.Printf("[Daemon] macOS service uninstalled successfully")
	return nil
}

// Start starts the installed service
func Start() error {
	if runtime.GOOS == "darwin" {
		return startDarwinService()
	}

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

func startDarwinService() error {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", ServiceName)

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return fmt.Errorf("服务未安装")
	}

	// Check if service is already running
	status, _ := darwinServiceStatus()
	if status == service.StatusRunning {
		log.Printf("[Daemon] Service is already running, skipping start")
		return nil
	}

	// Try launchctl kickstart first (doesn't require sudo for already-loaded services)
	// kickstart -k will restart a running service, or start a stopped one
	kickstartCmd := exec.Command("launchctl", "kickstart", "-k", fmt.Sprintf("system/%s", ServiceName))
	if output, err := kickstartCmd.CombinedOutput(); err == nil {
		log.Printf("[Daemon] Service started via kickstart")
		return nil
	} else {
		log.Printf("[Daemon] kickstart failed: %s, trying load with admin privileges", strings.TrimSpace(string(output)))
	}

	// Fallback to load with admin privileges (required if service was unloaded)
	script := fmt.Sprintf(`
do shell script "launchctl load -w '%s'" with administrator privileges
`, plistPath)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if strings.Contains(outputStr, "User canceled") || strings.Contains(outputStr, "-128") {
			return fmt.Errorf("用户取消了授权")
		}
		return fmt.Errorf("启动服务失败: %s", outputStr)
	}
	return nil
}

// Stop stops the installed service
func Stop() error {
	if runtime.GOOS == "darwin" {
		return stopDarwinService()
	}

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

func stopDarwinService() error {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", ServiceName)

	// Check if plist exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return fmt.Errorf("服务未安装")
	}

	// Check if service is already stopped
	status, _ := darwinServiceStatus()
	if status == service.StatusStopped {
		log.Printf("[Daemon] Service is already stopped, skipping stop")
		return nil
	}

	// Note: Due to KeepAlive: true in plist, the service will auto-restart after unload
	// To permanently stop, user should uninstall the service
	script := fmt.Sprintf(`
do shell script "launchctl unload '%s'" with administrator privileges
`, plistPath)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := strings.TrimSpace(string(output))
		if strings.Contains(outputStr, "User canceled") || strings.Contains(outputStr, "-128") {
			return fmt.Errorf("用户取消了授权")
		}
		return fmt.Errorf("停止服务失败: %s", outputStr)
	}
	return nil
}

// Restart restarts the installed service
func Restart() error {
	if runtime.GOOS == "darwin" {
		if err := stopDarwinService(); err != nil {
			// Ignore stop error, service might not be running
		}
		return startDarwinService()
	}

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
	if runtime.GOOS == "darwin" {
		return darwinServiceStatus()
	}

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

func darwinServiceStatus() (service.Status, error) {
	plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", ServiceName)

	// Check if plist exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		return service.StatusUnknown, nil
	}

	// Use launchctl print to check system service status (works without sudo)
	cmd := exec.Command("launchctl", "print", fmt.Sprintf("system/%s", ServiceName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Service is installed but not loaded
		return service.StatusStopped, nil
	}

	outputStr := string(output)
	// Check for "state = running" in output
	if strings.Contains(outputStr, "state = running") {
		return service.StatusRunning, nil
	}

	return service.StatusStopped, nil
}

// IsInstalled checks if the service is installed
func IsInstalled() bool {
	if runtime.GOOS == "darwin" {
		plistPath := fmt.Sprintf("/Library/LaunchDaemons/%s.plist", ServiceName)
		_, err := os.Stat(plistPath)
		return err == nil
	}

	status, err := Status()
	if err != nil {
		return false
	}
	return status != service.StatusUnknown
}

// GetInstalledServicePath returns the executable path of the installed service
// Returns empty string if service is not installed or path cannot be determined
func GetInstalledServicePath() string {
	if runtime.GOOS == "windows" {
		return getWindowsServicePath()
	}
	// For other platforms, not implemented yet
	return ""
}

// getWindowsServicePath queries Windows Service Manager to get the service executable path
func getWindowsServicePath() string {
	// Use sc qc command to query service configuration
	cmd := exec.Command("sc", "qc", ServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	// Parse output to find BINARY_PATH_NAME
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "BINARY_PATH_NAME") {
			// Format: BINARY_PATH_NAME   : "C:\path\to\pfm.exe" service run
			// or: BINARY_PATH_NAME   : C:\path\to\pfm.exe service run
			parts := strings.SplitN(line, ":", 2)
			if len(parts) < 2 {
				continue
			}
			pathPart := strings.TrimSpace(parts[1])

			// Extract the executable path (may be quoted)
			if strings.HasPrefix(pathPart, "\"") {
				// Quoted path
				endQuote := strings.Index(pathPart[1:], "\"")
				if endQuote > 0 {
					return pathPart[1 : endQuote+1]
				}
			} else {
				// Unquoted path - take until first space (before arguments)
				spaceIdx := strings.Index(pathPart, " ")
				if spaceIdx > 0 {
					return pathPart[:spaceIdx]
				}
				return pathPart
			}
		}
	}
	return ""
}

// NeedsReinstall checks if the service needs to be reinstalled
// Returns true if service is installed but points to a different executable path
func NeedsReinstall() bool {
	if !IsInstalled() {
		return false
	}

	installedPath := GetInstalledServicePath()
	if installedPath == "" {
		// Cannot determine installed path, assume no reinstall needed
		return false
	}

	currentPath, err := os.Executable()
	if err != nil {
		return false
	}
	currentPath, err = filepath.EvalSymlinks(currentPath)
	if err != nil {
		return false
	}

	// Normalize paths for comparison (case-insensitive on Windows)
	if runtime.GOOS == "windows" {
		installedPath = strings.ToLower(filepath.Clean(installedPath))
		currentPath = strings.ToLower(filepath.Clean(currentPath))
	} else {
		installedPath = filepath.Clean(installedPath)
		currentPath = filepath.Clean(currentPath)
	}

	return installedPath != currentPath
}

// IsRunning checks if the service is running
func IsRunning() bool {
	status, err := Status()
	if err != nil {
		return false
	}
	return status == service.StatusRunning
}
