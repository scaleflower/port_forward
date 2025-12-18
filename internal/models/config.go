package models

// AppConfig represents the application configuration
type AppConfig struct {
	// General settings
	LogLevel       string `json:"logLevel"`       // debug, info, warn, error
	AutoStart      bool   `json:"autoStart"`      // Start rules on app launch
	StartMinimized bool   `json:"startMinimized"` // Start minimized to tray

	// Tray settings
	TrayEnabled bool `json:"trayEnabled"` // Show system tray icon

	// Hotkey settings
	HotkeyEnabled   bool   `json:"hotkeyEnabled"`   // Enable global hotkey
	HotkeyModifiers string `json:"hotkeyModifiers"` // e.g., "cmd+shift", "ctrl+shift"
	HotkeyKey       string `json:"hotkeyKey"`       // e.g., "p", "f", "m"

	// Service settings
	ServiceEnabled bool `json:"serviceEnabled"` // Run as background service
	ServicePort    int  `json:"servicePort"`    // IPC port (default: 0 for auto)

	// API settings (for advanced users)
	APIEnabled bool   `json:"apiEnabled"`
	APIAddr    string `json:"apiAddr"` // e.g., ":18080"
	APIAuth    *Auth  `json:"apiAuth,omitempty"`

	// Metrics settings
	MetricsEnabled bool   `json:"metricsEnabled"`
	MetricsAddr    string `json:"metricsAddr"` // e.g., ":9000"
}

// DefaultAppConfig returns the default application configuration
func DefaultAppConfig() *AppConfig {
	return &AppConfig{
		LogLevel:        "info",
		AutoStart:       true,
		StartMinimized:  false,
		TrayEnabled:     true,               // Enable tray by default
		HotkeyEnabled:   true,               // Enable hotkey by default
		HotkeyModifiers: "cmd+shift",        // Default: Cmd+Shift on macOS
		HotkeyKey:       "p",                // Default: P key
		ServiceEnabled:  false,
		ServicePort:     0,
		APIEnabled:      false,
		APIAddr:         ":18080",
		MetricsEnabled:  false,
		MetricsAddr:     ":9000",
	}
}

// AppData represents all persistent application data
type AppData struct {
	Config *AppConfig `json:"config"`
	Rules  []*Rule    `json:"rules"`
	Chains []*Chain   `json:"chains"`
}

// NewAppData creates a new AppData with default config
func NewAppData() *AppData {
	return &AppData{
		Config: DefaultAppConfig(),
		Rules:  []*Rule{},
		Chains: []*Chain{},
	}
}

// ServiceStatus represents the status of the background service
type ServiceStatus struct {
	Running     bool   `json:"running"`
	PID         int    `json:"pid,omitempty"`
	StartTime   string `json:"startTime,omitempty"`
	RulesActive int    `json:"rulesActive"`
	RulesTotal  int    `json:"rulesTotal"`
	Version     string `json:"version"`
}

// RuleStats represents statistics for a rule
type RuleStats struct {
	RuleID       string `json:"ruleId"`
	BytesIn      int64  `json:"bytesIn"`
	BytesOut     int64  `json:"bytesOut"`
	Connections  int64  `json:"connections"`
	ActiveConns  int    `json:"activeConns"`
	Errors       int64  `json:"errors"`
	LastActivity string `json:"lastActivity,omitempty"`
}
