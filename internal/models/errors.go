package models

import (
	"errors"
	"fmt"
)

// Common errors
var (
	// Rule errors
	ErrRuleNameEmpty   = errors.New("rule name cannot be empty")
	ErrListenAddrEmpty = errors.New("listen address cannot be empty")
	ErrNoTargets       = errors.New("at least one target is required")
	ErrRuleNotFound    = errors.New("rule not found")
	ErrRuleExists      = errors.New("rule already exists")

	// Chain errors
	ErrChainNameEmpty = errors.New("chain name cannot be empty")
	ErrNoHops         = errors.New("at least one hop is required")
	ErrChainNotFound  = errors.New("chain not found")
	ErrChainExists    = errors.New("chain already exists")
	ErrChainInUse     = errors.New("chain is in use by one or more rules")

	// Service errors
	ErrServiceNotRunning = errors.New("service is not running")
	ErrServiceRunning    = errors.New("service is already running")
	ErrServiceStart      = errors.New("failed to start service")
	ErrServiceStop       = errors.New("failed to stop service")
)

// ValidationError represents a validation error with field details
type ValidationError struct {
	Field   string
	Index   int
	Message string
}

func (e *ValidationError) Error() string {
	if e.Index >= 0 {
		return fmt.Sprintf("validation error on %s[%d]: %s", e.Field, e.Index, e.Message)
	}
	return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
}

// EngineError represents an engine-related error
type EngineError struct {
	RuleID  string
	Op      string
	Message string
	Err     error
}

func (e *EngineError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("engine error [%s] on rule %s: %s (%v)", e.Op, e.RuleID, e.Message, e.Err)
	}
	return fmt.Sprintf("engine error [%s] on rule %s: %s", e.Op, e.RuleID, e.Message)
}

func (e *EngineError) Unwrap() error {
	return e.Err
}
