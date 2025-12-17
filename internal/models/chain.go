package models

import (
	"time"

	"github.com/google/uuid"
)

// Chain represents a proxy chain configuration
type Chain struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Hops        []Hop     `json:"hops"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Hop represents a single hop in the proxy chain
type Hop struct {
	Name     string   `json:"name"`
	Addr     string   `json:"addr"`     // e.g., "proxy.example.com:1080"
	Protocol Protocol `json:"protocol"` // socks5, http, ss, etc.
	Auth     *Auth    `json:"auth,omitempty"`
	TLS      *TLSConfig `json:"tls,omitempty"`
}

// NewChain creates a new chain with default values
func NewChain(name string) *Chain {
	now := time.Now()
	return &Chain{
		ID:        uuid.New().String(),
		Name:      name,
		Hops:      []Hop{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddHop adds a hop to the chain
func (c *Chain) AddHop(hop Hop) {
	c.Hops = append(c.Hops, hop)
	c.UpdatedAt = time.Now()
}

// RemoveHop removes a hop from the chain by index
func (c *Chain) RemoveHop(index int) {
	if index >= 0 && index < len(c.Hops) {
		c.Hops = append(c.Hops[:index], c.Hops[index+1:]...)
		c.UpdatedAt = time.Now()
	}
}

// Validate validates the chain configuration
func (c *Chain) Validate() error {
	if c.Name == "" {
		return ErrChainNameEmpty
	}
	if len(c.Hops) == 0 {
		return ErrNoHops
	}
	for i, hop := range c.Hops {
		if hop.Addr == "" {
			return &ValidationError{Field: "hops", Index: i, Message: "hop address is empty"}
		}
	}
	return nil
}

// Clone creates a deep copy of the chain
func (c *Chain) Clone() *Chain {
	clone := *c
	clone.Hops = make([]Hop, len(c.Hops))
	for i, hop := range c.Hops {
		clone.Hops[i] = hop
		if hop.Auth != nil {
			auth := *hop.Auth
			clone.Hops[i].Auth = &auth
		}
		if hop.TLS != nil {
			tls := *hop.TLS
			clone.Hops[i].TLS = &tls
		}
	}
	return &clone
}
