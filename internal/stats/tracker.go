package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/y0ug/ai-helper/internal/io"
)

// Tracker manages statistics recording and persistence
type Tracker struct {
	stats    *Stats
	filePath string
}

// NewTracker creates a new statistics tracker
func NewTracker() (*Tracker, error) {
	cacheDir := io.GetCacheDir()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	t := &Tracker{
		stats:    NewStats(),
		filePath: filepath.Join(cacheDir, "stats.json"),
	}

	// Load existing stats if available
	if err := t.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load stats: %w", err)
	}

	return t, nil
}

// RecordQuery records statistics for a single query
func (t *Tracker) RecordQuery(provider string, command string, inputTokens, outputTokens int, cost float64, functionCalls int) {
	t.stats.mu.Lock()
	defer t.stats.mu.Unlock()

	if _, exists := t.stats.Providers[provider]; !exists {
		t.stats.Providers[provider] = &ProviderStats{
			Commands: make(map[string]*CommandStats),
		}
	}

	stats := t.stats.Providers[provider]
	stats.Queries++
	stats.InputTokens += int64(inputTokens)
	stats.OutputTokens += int64(outputTokens)
	stats.Cost += cost
	stats.LastUsed = time.Now()

	// Update command-specific stats
	if command != "" {
		if _, exists := stats.Commands[command]; !exists {
			stats.Commands[command] = &CommandStats{}
		}
		cmdStats := stats.Commands[command]
		cmdStats.Count++
		cmdStats.InputTokens += int64(inputTokens)
		cmdStats.OutputTokens += int64(outputTokens)
		cmdStats.Cost += cost
		cmdStats.LastUsed = time.Now()
	}

	// Save after each update
	if err := t.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save stats: %v\n", err)
	}
}

// GetStats returns a copy of the current statistics
func (t *Tracker) GetStats() map[string]ProviderStats {
	t.stats.mu.RLock()
	defer t.stats.mu.RUnlock()

	stats := make(map[string]ProviderStats)
	for provider, providerStats := range t.stats.Providers {
		stats[provider] = *providerStats
	}
	return stats
}

// Save persists the current statistics to disk
func (t *Tracker) Save() error {
	data, err := json.MarshalIndent(t.stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	if err := os.WriteFile(t.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write stats file: %w", err)
	}

	return nil
}

// Load reads statistics from disk
func (t *Tracker) Load() error {
	data, err := os.ReadFile(t.filePath)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, t.stats); err != nil {
		return fmt.Errorf("failed to unmarshal stats: %w", err)
	}

	return nil
}
