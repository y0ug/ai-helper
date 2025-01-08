package ai

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	lastUpdate    time.Time
	infoURL       string
	infoFile      string
	configDir     string
	cacheDuration time.Duration
}

func NewInfoProviders(configDir string) *InfoProviders {
	return &InfoProviders{
		infos:         make(map[string]Info),
		infoURL:       "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json",
		infoFile:      "model_prices_and_context_window.json",
		cacheDuration: 24 * time.Hour,
		configDir:     configDir,
	}
}

func (t *InfoProviders) Clear() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clear the in-memory cache
	t.infos = make(map[string]Info)

	// Reset the last update time
	t.lastUpdate = time.Time{}

	// Delete the cached file if it exists
	infoPath := filepath.Join(t.configDir, ".config", "ai-helper", t.infoFile)
	if _, err := os.Stat(infoPath); err == nil {
		if err := os.Remove(infoPath); err != nil {
			return fmt.Errorf("failed to remove info file: %w", err)
		}
	}

	return nil
}

func (t *InfoProviders) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// configDir, _ := os.UserHomeDir()
	infoPath := filepath.Join(t.configDir, ".config", "ai-helper", t.infoFile)

	// Check if we need to update the pricing file
	needsUpdate := true
	if info, err := os.Stat(infoPath); err == nil {
		t.lastUpdate = info.ModTime()
		if time.Since(t.lastUpdate) < t.cacheDuration {
			needsUpdate = false
		}
	}

	if needsUpdate {
		if err := t.downloadInfo(infoPath); err != nil {
			return fmt.Errorf("failed to download pricing: %w", err)
		}
	}

	data, err := os.ReadFile(infoPath)
	if err != nil {
		return fmt.Errorf("failed to read pricing file: %w", err)
	}

	if err := json.Unmarshal(data, &t.infos); err != nil {
		return fmt.Errorf("failed to parse pricing data: %w", err)
	}

	return nil
}

func (t *InfoProviders) downloadInfo(infoPath string) error {
	resp, err := http.Get(t.infoURL)
	if err != nil {
		return fmt.Errorf("failed to download pricing data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download pricing data: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read pricing data: %w", err)
	}

	if err := os.WriteFile(infoPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write pricing file: %w", err)
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
