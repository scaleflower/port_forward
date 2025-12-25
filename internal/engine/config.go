package engine

import (
	"encoding/json"
	"fmt"
	"log"

	"pfm/internal/models"

	"github.com/go-gost/x/config"
)

// RuleToGostConfig converts a Rule to gost configuration
func RuleToGostConfig(rule *models.Rule, chains []*models.Chain) (*config.Config, error) {
	cfg := &config.Config{
		Services: []*config.ServiceConfig{},
		Chains:   []*config.ChainConfig{},
	}

	// Build service configuration
	svc, err := buildServiceConfig(rule)
	if err != nil {
		return nil, err
	}
	cfg.Services = append(cfg.Services, svc)

	// Build chain configuration if referenced
	if rule.ChainID != "" {
		var chain *models.Chain
		for _, c := range chains {
			if c.ID == rule.ChainID {
				chain = c
				break
			}
		}
		if chain != nil {
			chainCfg, err := buildChainConfig(chain)
			if err != nil {
				return nil, err
			}
			cfg.Chains = append(cfg.Chains, chainCfg)
			svc.Handler.Chain = chain.ID
		}
	}

	// Debug: print full config JSON
	if cfgJSON, err := json.MarshalIndent(cfg, "", "  "); err == nil {
		log.Printf("[Config] Full gost config for rule %s:\n%s", rule.ID, string(cfgJSON))
	}

	return cfg, nil
}

// buildServiceConfig creates a gost service configuration from a rule
func buildServiceConfig(rule *models.Rule) (*config.ServiceConfig, error) {
	svc := &config.ServiceConfig{
		Name:     rule.ID,
		Addr:     rule.GetListenAddr(),
		Observer: observerName, // Use our stats observer
		Metadata: map[string]any{
			"enableStats":     true, // Enable statistics tracking for observer
			"observer.period": "3s", // Report stats every 3 seconds
		},
	}

	// Debug: log the service config
	log.Printf("[Config] Building service config for rule %s: Observer=%s, enableStats=%v",
		rule.ID, svc.Observer, svc.Metadata["enableStats"])

	// Configure handler based on rule type
	switch rule.Type {
	case models.RuleTypeForward:
		svc.Handler = buildForwardHandler(rule)
		svc.Listener = buildListener(rule)
		svc.Forwarder = buildForwarder(rule)

	case models.RuleTypeReverse:
		svc.Handler = buildReverseHandler(rule)
		svc.Listener = buildListener(rule)
		svc.Forwarder = buildForwarder(rule)

	case models.RuleTypeChain:
		// Chain type uses proxy protocols
		svc.Handler = buildProxyHandler(rule)
		svc.Listener = buildListener(rule)

	default:
		return nil, fmt.Errorf("unsupported rule type: %s", rule.Type)
	}

	// Add authentication if configured
	if rule.Auth != nil && rule.Auth.Username != "" {
		svc.Handler.Auth = &config.AuthConfig{
			Username: rule.Auth.Username,
			Password: rule.Auth.Password,
		}
	}

	return svc, nil
}

// buildForwardHandler creates a local forward handler configuration
// Note: forward/tcp handler doesn't support observer, stats are tracked at listener level
func buildForwardHandler(rule *models.Rule) *config.HandlerConfig {
	// In Gost v3, "forward" handler handles both TCP and UDP forwarding
	handlerType := "forward"

	// Just for explicitness, though both map to "forward"
	switch rule.Protocol {
	case models.ProtocolUDP:
		handlerType = "forward"
	case models.ProtocolHTTP, models.ProtocolHTTPS:
		handlerType = "forward"
	default:
		handlerType = "forward"
	}

	return &config.HandlerConfig{
		Type: handlerType,
	}
}

// buildReverseHandler creates a remote forward handler configuration
func buildReverseHandler(rule *models.Rule) *config.HandlerConfig {
	return &config.HandlerConfig{
		Type: "forward",
	}
}

// buildProxyHandler creates a proxy handler configuration
func buildProxyHandler(rule *models.Rule) *config.HandlerConfig {
	handlerType := "socks5"
	switch rule.Protocol {
	case models.ProtocolHTTP, models.ProtocolHTTPS:
		handlerType = "http"
	case models.ProtocolSOCKS5:
		handlerType = "socks5"
	case models.ProtocolSS:
		handlerType = "ss"
	default:
		handlerType = "socks5"
	}

	return &config.HandlerConfig{
		Type:     handlerType,
		Observer: observerName, // Handler-level observer for request stats
	}
}

// buildListener creates a listener configuration
func buildListener(rule *models.Rule) *config.ListenerConfig {
	listenerType := "tcp"
	switch rule.Protocol {
	case models.ProtocolUDP:
		listenerType = "udp"
	default:
		listenerType = "tcp"
	}

	listener := &config.ListenerConfig{
		Type: listenerType,
	}

	// Add TLS configuration if enabled
	if rule.TLS != nil && rule.TLS.Enabled {
		listener.TLS = &config.TLSConfig{
			CertFile:   rule.TLS.CertFile,
			KeyFile:    rule.TLS.KeyFile,
			CAFile:     rule.TLS.CAFile,
			Secure:     rule.TLS.Secure,
			ServerName: rule.TLS.ServerName,
		}
	}

	return listener
}

// buildForwarder creates a forwarder configuration with targets
func buildForwarder(rule *models.Rule) *config.ForwarderConfig {
	var nodes []*config.ForwardNodeConfig

	// Check simple mode (single target)
	if rule.TargetHost != "" && rule.TargetPort > 0 {
		nodes = []*config.ForwardNodeConfig{
			{
				Name: "target-0",
				Addr: rule.GetTargetAddr(),
			},
		}
	} else if len(rule.Targets) > 0 {
		// Multi-target mode (load balancing)
		nodes = make([]*config.ForwardNodeConfig, len(rule.Targets))
		for i, target := range rule.Targets {
			nodes[i] = &config.ForwardNodeConfig{
				Name: fmt.Sprintf("target-%d", i),
				Addr: fmt.Sprintf("%s:%d", target.Host, target.Port),
			}
		}
	} else {
		return nil
	}

	forwarder := &config.ForwarderConfig{
		Nodes: nodes,
	}

	// Add selector for load balancing if multiple targets
	if len(nodes) > 1 {
		forwarder.Selector = &config.SelectorConfig{
			Strategy: "round",
		}
	}

	return forwarder
}

// buildChainConfig creates a gost chain configuration
func buildChainConfig(chain *models.Chain) (*config.ChainConfig, error) {
	cfg := &config.ChainConfig{
		Name: chain.ID,
		Hops: []*config.HopConfig{},
	}

	for i, hop := range chain.Hops {
		hopCfg := &config.HopConfig{
			Name: fmt.Sprintf("hop-%d", i),
			Nodes: []*config.NodeConfig{
				{
					Name:      hop.Name,
					Addr:      hop.Addr,
					Connector: buildConnector(hop),
					Dialer:    buildDialer(hop),
				},
			},
		}
		cfg.Hops = append(cfg.Hops, hopCfg)
	}

	return cfg, nil
}

// buildConnector creates a connector configuration for a hop
func buildConnector(hop models.Hop) *config.ConnectorConfig {
	connectorType := "socks5"
	switch hop.Protocol {
	case models.ProtocolHTTP, models.ProtocolHTTPS:
		connectorType = "http"
	case models.ProtocolSOCKS5:
		connectorType = "socks5"
	case models.ProtocolSS:
		connectorType = "ss"
	default:
		connectorType = "socks5"
	}

	connector := &config.ConnectorConfig{
		Type: connectorType,
	}

	if hop.Auth != nil && hop.Auth.Username != "" {
		connector.Auth = &config.AuthConfig{
			Username: hop.Auth.Username,
			Password: hop.Auth.Password,
		}
	}

	return connector
}

// buildDialer creates a dialer configuration for a hop
func buildDialer(hop models.Hop) *config.DialerConfig {
	dialerType := "tcp"

	dialer := &config.DialerConfig{
		Type: dialerType,
	}

	if hop.TLS != nil && hop.TLS.Enabled {
		dialer.TLS = &config.TLSConfig{
			Secure:     hop.TLS.Secure,
			ServerName: hop.TLS.ServerName,
		}
	}

	return dialer
}
