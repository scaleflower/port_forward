package engine

import (
	"testing"

	"pfm/internal/models"
)

func TestRuleToGostConfig(t *testing.T) {
	tests := []struct {
		name    string
		rule    *models.Rule
		chains  []*models.Chain
		wantErr bool
		check   func(*testing.T, *models.Rule) // Helper to check generated config details if we could return it,
		// but RuleToGostConfig returns *config.Config (gost).
		// Since we can't easily import internal gost structures here without cycle or complex setup,
		// we mainly check for error and rely on logical flow.
		// However, engine package imports gost/x/config, so we can check it.
	}{
		{
			name: "Forward TCP Rule",
			rule: &models.Rule{
				ID:         "rule-1",
				Name:       "Forward TCP",
				Type:       models.RuleTypeForward,
				Protocol:   models.ProtocolTCP,
				LocalPort:  8080,
				TargetHost: "127.0.0.1",
				TargetPort: 80,
			},
			wantErr: false,
		},
		{
			name: "Forward UDP Rule",
			rule: &models.Rule{
				ID:         "rule-2",
				Name:       "Forward UDP",
				Type:       models.RuleTypeForward,
				Protocol:   models.ProtocolUDP,
				LocalPort:  5353,
				TargetHost: "8.8.8.8",
				TargetPort: 53,
			},
			wantErr: false,
		},
		{
			name: "Chain Rule SOCKS5",
			rule: &models.Rule{
				ID:        "rule-3",
				Name:      "SOCKS5 Proxy",
				Type:      models.RuleTypeChain,
				Protocol:  models.ProtocolSOCKS5,
				LocalPort: 1080,
			},
			wantErr: false,
		},
		{
			name: "Forward with Chain",
			rule: &models.Rule{
				ID:         "rule-4",
				Name:       "Forward Over Chain",
				Type:       models.RuleTypeForward,
				Protocol:   models.ProtocolTCP,
				LocalPort:  8081,
				TargetHost: "1.1.1.1",
				TargetPort: 80,
				ChainID:    "chain-1",
			},
			chains: []*models.Chain{
				{
					ID:   "chain-1",
					Name: "MyChain",
					Hops: []models.Hop{
						{Name: "hop1", Addr: "192.168.1.1:1080", Protocol: models.ProtocolSOCKS5},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := RuleToGostConfig(tt.rule, tt.chains)
			if (err != nil) != tt.wantErr {
				t.Errorf("RuleToGostConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && cfg != nil {
				if len(cfg.Services) != 1 {
					t.Errorf("Expected 1 service, got %d", len(cfg.Services))
				}
				svc := cfg.Services[0]
				if svc.Name != tt.rule.ID {
					t.Errorf("Service name mismatch, got %s, want %s", svc.Name, tt.rule.ID)
				}
				// Verify Chain if present
				if tt.rule.ChainID != "" {
					if len(cfg.Chains) != 1 {
						t.Errorf("Expected 1 chain config, got %d", len(cfg.Chains))
					} else {
						if cfg.Chains[0].Name != tt.rule.ChainID {
							t.Errorf("Chain name mismatch, got %s, want %s", cfg.Chains[0].Name, tt.rule.ChainID)
						}
					}
				}
			}
		})
	}
}
