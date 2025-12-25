package models

import (
	"testing"
)

func TestRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    *Rule
		wantErr bool
	}{
		{
			name: "Valid Forward Rule (Simple)",
			rule: &Rule{
				Name:       "Test Rule",
				LocalPort:  8080,
				Type:       RuleTypeForward,
				TargetHost: "127.0.0.1",
				TargetPort: 80,
			},
			wantErr: false,
		},
		{
			name: "Valid Forward Rule (Multi-target)",
			rule: &Rule{
				Name:      "Test Rule",
				LocalPort: 8080,
				Type:      RuleTypeForward,
				Targets: []Target{
					{Host: "127.0.0.1", Port: 80},
					{Host: "127.0.0.1", Port: 81},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid Chain Rule",
			rule: &Rule{
				Name:      "Test Chain",
				LocalPort: 1080,
				Type:      RuleTypeChain,
				// Targets optional for chain type if it's just a proxy server
			},
			wantErr: false,
		},
		{
			name: "Invalid - Empty Name",
			rule: &Rule{
				Name:       "",
				LocalPort:  8080,
				TargetHost: "127.0.0.1",
				TargetPort: 80,
			},
			wantErr: true,
		},
		{
			name: "Invalid - Zero Local Port",
			rule: &Rule{
				Name:       "Test",
				LocalPort:  0,
				TargetHost: "127.0.0.1",
				TargetPort: 80,
			},
			wantErr: true,
		},
		{
			name: "Invalid - No Targets (Forward)",
			rule: &Rule{
				Name:      "Test",
				LocalPort: 8080,
				Type:      RuleTypeForward,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rule.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Rule.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewRule(t *testing.T) {
	rule := NewRule("MyRule", RuleTypeForward)

	if rule.Name != "MyRule" {
		t.Errorf("NewRule().Name = %v, want %v", rule.Name, "MyRule")
	}
	if rule.Type != RuleTypeForward {
		t.Errorf("NewRule().Type = %v, want %v", rule.Type, RuleTypeForward)
	}
	if rule.ID == "" {
		t.Error("NewRule().ID should not be empty")
	}
	if rule.Status != RuleStatusStopped {
		t.Errorf("NewRule().Status = %v, want %v", rule.Status, RuleStatusStopped)
	}
	if !rule.CreatedAt.Equal(rule.UpdatedAt) {
		t.Error("NewRule() CreatedAt should equal UpdatedAt")
	}
}
