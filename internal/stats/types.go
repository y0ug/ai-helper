package stats

import (
	"sync"
	"time"
)

// CommandStats holds statistics for a single command
type CommandStats struct {
	Count        int64     `json:"count"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	Cost         float64   `json:"cost"`
	LastUsed     time.Time `json:"last_used"`
}

// ProviderStats holds statistics for a single provider
type ProviderStats struct {
	Queries       int64                    `json:"queries"`
	InputTokens   int64                    `json:"input_tokens"`
	OutputTokens  int64                    `json:"output_tokens"`
	Cost          float64                  `json:"cost"`
	LastUsed      time.Time                `json:"last_used"`
	Commands      map[string]*CommandStats `json:"commands"`
}

// Stats holds statistics for all providers
type Stats struct {
	Providers map[string]*ProviderStats `json:"providers"`
	mu        sync.RWMutex             `json:"-"`
}

// NewStats creates a new Stats instance
func NewStats() *Stats {
	return &Stats{
		Providers: make(map[string]*ProviderStats),
	}
}
