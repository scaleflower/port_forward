package engine

import (
	"fmt"
	"log"

	"pfm/internal/models"

	"github.com/go-gost/core/logger"
	"github.com/go-gost/core/service"
	chain_parser "github.com/go-gost/x/config/parsing/chain"
	service_parser "github.com/go-gost/x/config/parsing/service"
	"github.com/go-gost/x/registry"
)

// BuildService builds a GOST service from a Rule and Chains
func BuildService(rule *models.Rule, chains []*models.Chain) (service.Service, error) {
	// Convert rule to gost config
	cfg, err := RuleToGostConfig(rule, chains)
	if err != nil {
		return nil, fmt.Errorf("failed to build configuration: %w", err)
	}

	// Parse the service configuration directly
	if len(cfg.Services) == 0 {
		return nil, fmt.Errorf("no service configuration generated")
	}

	svcCfg := cfg.Services[0]
	log.Printf("[Builder] Building service for rule %s: Addr=%s, Handler=%s, Listener=%s",
		rule.ID, svcCfg.Addr, svcCfg.Handler.Type, svcCfg.Listener.Type)

	// Parse and register chain if present
	if len(cfg.Chains) > 0 {
		for _, chainCfg := range cfg.Chains {
			log.Printf("[Builder] Parsing chain: %s", chainCfg.Name)
			chain, err := chain_parser.ParseChain(chainCfg, logger.Default())
			if err != nil {
				return nil, fmt.Errorf("failed to parse chain %s: %w", chainCfg.Name, err)
			}
			// Register the chain so service_parser can resolve it
			if err := registry.ChainRegistry().Register(chainCfg.Name, chain); err != nil {
				log.Printf("[Builder] Chain %s already registered or error: %v", chainCfg.Name, err)
			} else {
				log.Printf("[Builder] Chain registered: %s", chainCfg.Name)
			}
		}
	}

	// Create the service using service_parser.ParseService
	svc, err := service_parser.ParseService(svcCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
	}

	if svc == nil {
		return nil, fmt.Errorf("service is nil")
	}

	// Register the service in the registry for later access
	if err := registry.ServiceRegistry().Register(rule.ID, svc); err != nil {
		return nil, fmt.Errorf("failed to register service: %w", err)
	}

	log.Printf("[Builder] Service created and registered: %s, listening on %s", rule.ID, svc.Addr().String())
	return svc, nil
}
