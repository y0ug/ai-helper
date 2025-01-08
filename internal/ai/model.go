package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Model represents an AI model configuration
type Model struct {
	Provider string
	Name     string
	info     *Info
}

type Info struct {
	MaxTokens                 int
	MaxInputTokens            int
	MaxOutputTokens           int
	InputCostPerToken         float64
	OutputCostPerToken        float64
	LiteLLMProvider           string
	Mode                      string
	SupportsFunctionCalling   bool
	SupportsVision            bool
	ToolUseSystemPromptTokens int
	SupportsAssistantPrefill  bool
	SupportsPromptCaching     bool
	SupportsResponseSchema    bool
}

type InfoProviders struct {
	mu            sync.RWMutex
	infos         map[string]Info
	infoURL       string
	infoFile      string
	cacheFile     string
	lastUpdate    time.Time
	cacheDuration time.Duration
}

func NewInfoProviders(infoFilePath string) *InfoProviders {
	info := &InfoProviders{
		infos:         make(map[string]Info),
		infoURL:       "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json",
		infoFile:      "model_prices_and_context_window.json",
		cacheFile:     infoFilePath,
		cacheDuration: 24 * time.Hour,
	}
	info.Load()
	return info
}

func (t *InfoProviders) Clear() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clear the in-memory cache
	t.infos = make(map[string]Info)

	// Delete the cached file if it exists
	if t.cacheFile != "" {
		if _, err := os.Stat(t.cacheFile); err == nil {
			if err := os.Remove(t.cacheFile); err != nil {
				return fmt.Errorf("failed to remove info file: %w", err)
			}
		}
	}

	return nil
}

func (t *InfoProviders) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// If no cache file is specified, just download fresh data
	if t.cacheFile == "" {
		return t.downloadToMemory()
	}

	// Check if we need to update the cache file
	needsUpdate := true
	if info, err := os.Stat(t.cacheFile); err == nil {
		t.lastUpdate = info.ModTime()
		if time.Since(t.lastUpdate) < t.cacheDuration {
			needsUpdate = false
		}
	}

	if needsUpdate {
		// Download and save fresh data
		return t.downloadInfo(t.cacheFile)
	}

	// Load from cache file
	data, err := os.ReadFile(t.cacheFile)
	if err != nil {
		return fmt.Errorf("failed to read info file: %w", err)
	}

	if err := json.Unmarshal(data, &t.infos); err != nil {
		return fmt.Errorf("failed to parse info data: %w", err)
	}

	return nil
}

func (t *InfoProviders) downloadToMemory() error {
	resp, err := http.Get(t.infoURL)
	if err != nil {
		return fmt.Errorf("failed to download info data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download info data: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read info data: %w", err)
	}

	if err := json.Unmarshal(data, &t.infos); err != nil {
		return fmt.Errorf("failed to parse info data: %w", err)
	}

	return nil
}

func (t *InfoProviders) downloadInfo(infoPath string) error {
	if err := t.downloadToMemory(); err != nil {
		return err
	}

	// Save to cache file
	data, err := json.Marshal(t.infos)
	if err != nil {
		return fmt.Errorf("failed to marshal info data: %w", err)
	}

	if err := os.WriteFile(infoPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write info file: %w", err)
	}

	return nil
}

func (t *InfoProviders) GetModelInfo(model string) (*Info, error) {
	// Try the full model name first
	if data, ok := t.infos[model]; ok {
		return &data, nil
	}

	// Try with just the model name without provider prefix
	parts := strings.Split(model, "/")
	if len(parts) > 1 {
		modelName := parts[len(parts)-1]
		if data, ok := t.infos[modelName]; ok {
			return &data, nil
		}
	}

	return nil, fmt.Errorf("no info data for model: %s", model)
}

// ParseModel parses a model string in the format "provider/model"
func ParseModel(modelStr string, infoProviders *InfoProviders) (*Model, error) {
	parts := strings.Split(modelStr, "/")

	if len(parts) < 2 {
		return nil, fmt.Errorf(
			"invalid model format: %s, expected format: provider/model",
			modelStr,
		)
	}

	provider := parts[0]
	name := strings.Join(parts[1:], "/")

	var info2 *Info
	if infoProviders != nil {
		info, err := infoProviders.GetModelInfo(modelStr)
		if err != nil {
			fmt.Println(err)
		}
		info2 = info
	}
	return &Model{
		Provider: provider,
		Name:     name,
		info:     info2,
	}, nil
}

// String returns the string representation of the model
func (m *Model) String() string {
	return fmt.Sprintf("%s/%s", m.Provider, m.Name)
}
