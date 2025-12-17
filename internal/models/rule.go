package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RuleType represents the type of forwarding rule
type RuleType string

const (
	RuleTypeForward RuleType = "forward" // Port forwarding
	RuleTypeReverse RuleType = "reverse" // Reverse proxy
	RuleTypeChain   RuleType = "chain"   // Proxy chain
)

// RuleStatus represents the current status of a rule
type RuleStatus string

const (
	RuleStatusStopped RuleStatus = "stopped"
	RuleStatusRunning RuleStatus = "running"
	RuleStatusError   RuleStatus = "error"
)

// Protocol represents the network protocol
type Protocol string

const (
	ProtocolTCP    Protocol = "tcp"
	ProtocolUDP    Protocol = "udp"
	ProtocolHTTP   Protocol = "http"
	ProtocolHTTPS  Protocol = "https"
	ProtocolSOCKS5 Protocol = "socks5"
	ProtocolSS     Protocol = "ss" // Shadowsocks
)

// Environment represents the deployment environment
type Environment string

const (
	EnvTrunk      Environment = "TRUNK"
	EnvPreProd    Environment = "PRE-PROD"
	EnvProduction Environment = "PRODUCTION"
	EnvCustom     Environment = "CUSTOM"
)

// Rule represents a forwarding rule
type Rule struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`        // 用途
	Environment Environment `json:"environment"` // 环境
	Type        RuleType    `json:"type"`
	Enabled     bool        `json:"enabled"`
	LocalPort   int         `json:"localPort"`   // 本地映射端口
	Protocol    Protocol    `json:"protocol"`
	TargetHost  string      `json:"targetHost"`  // 目标 IP/域名
	TargetPort  int         `json:"targetPort"`  // 目标端口
	Targets     []Target    `json:"targets"`     // 保留用于负载均衡场景
	ChainID     string      `json:"chainId,omitempty"`
	Auth        *Auth       `json:"auth,omitempty"`
	TLS         *TLSConfig  `json:"tls,omitempty"`
	Status      RuleStatus  `json:"status"`
	ErrorMsg    string      `json:"errorMsg,omitempty"`
	Description string      `json:"description,omitempty"` // 用途描述
	Remark      string      `json:"remark,omitempty"`      // 备注
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

// Target represents a forwarding target (for load balancing)
type Target struct {
	Host   string `json:"host"`   // IP or hostname
	Port   int    `json:"port"`   // Port number
	Weight int    `json:"weight"` // Load balancing weight (default: 1)
}

// Auth represents authentication configuration
type Auth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled    bool   `json:"enabled"`
	CertFile   string `json:"certFile,omitempty"`
	KeyFile    string `json:"keyFile,omitempty"`
	CAFile     string `json:"caFile,omitempty"`
	ServerName string `json:"serverName,omitempty"`
	Secure     bool   `json:"secure"` // Verify server certificate
}

// NewRule creates a new rule with default values
func NewRule(name string, ruleType RuleType) *Rule {
	now := time.Now()
	return &Rule{
		ID:          uuid.New().String(),
		Name:        name,
		Type:        ruleType,
		Environment: EnvCustom,
		Enabled:     false,
		Protocol:    ProtocolTCP,
		Targets:     []Target{},
		Status:      RuleStatusStopped,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// Validate validates the rule configuration
func (r *Rule) Validate() error {
	if r.Name == "" {
		return ErrRuleNameEmpty
	}
	if r.LocalPort <= 0 {
		return ErrListenAddrEmpty
	}
	// Check simple mode (single target)
	if r.TargetHost != "" && r.TargetPort > 0 {
		return nil
	}
	// Check multi-target mode (load balancing)
	if len(r.Targets) == 0 && r.Type != RuleTypeChain {
		return ErrNoTargets
	}
	return nil
}

// GetListenAddr returns the listen address string
func (r *Rule) GetListenAddr() string {
	return fmt.Sprintf(":%d", r.LocalPort)
}

// GetTargetAddr returns the primary target address string
func (r *Rule) GetTargetAddr() string {
	if r.TargetHost != "" && r.TargetPort > 0 {
		return fmt.Sprintf("%s:%d", r.TargetHost, r.TargetPort)
	}
	if len(r.Targets) > 0 {
		return fmt.Sprintf("%s:%d", r.Targets[0].Host, r.Targets[0].Port)
	}
	return ""
}

// Clone creates a deep copy of the rule
func (r *Rule) Clone() *Rule {
	clone := *r
	clone.Targets = make([]Target, len(r.Targets))
	copy(clone.Targets, r.Targets)
	if r.Auth != nil {
		auth := *r.Auth
		clone.Auth = &auth
	}
	if r.TLS != nil {
		tls := *r.TLS
		clone.TLS = &tls
	}
	return &clone
}
