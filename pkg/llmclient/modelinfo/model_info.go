package modelinfo

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Model struct {
	Provider string
	Name     string
	Metadata *Metadata
}

type Metadata struct {
	// Required fields that should always be present
	MaxTokens          int     `json:"max_tokens"`
	MaxInputTokens     int     `json:"max_input_tokens"`
	MaxOutputTokens    int     `json:"max_output_tokens"`
	InputCostPerToken  float64 `json:"input_cost_per_token"`
	OutputCostPerToken float64 `json:"output_cost_per_token"`
	LiteLLMProvider    string  `json:"litellm_provider"`
	Mode               string  `json:"mode"`

	// Optional fields
	CacheCreationInputTokenCost float64 `json:"cache_creation_input_token_cost,omitempty"`
	CacheReadInputTokenCost     float64 `json:"cache_read_input_token_cost,omitempty"`
	ToolUseSystemPromptTokens   int     `json:"tool_use_system_prompt_tokens,omitempty"`

	// Optional feature flags
	SupportsFunctionCalling         bool `json:"supports_function_calling,omitempty"`
	SupportsParallelFunctionCalling bool `json:"supports_parallel_function_calling,omitempty"`
	SupportsVision                  bool `json:"supports_vision,omitempty"`
	SupportsAudioInput              bool `json:"supports_audio_input,omitempty"`
	SupportsAudioOutput             bool `json:"supports_audio_output,omitempty"`
	SupportsPromptCaching           bool `json:"supports_prompt_caching,omitempty"`
	SupportsResponseSchema          bool `json:"supports_response_schema,omitempty"`
	SupportsSystemMessages          bool `json:"supports_system_messages,omitempty"`
	SupportsAssistantPrefill        bool `json:"supports_assistant_prefill,omitempty"`
}

type Provider interface {
	Load() error
	Clear() error
	Get(modelName string) (*Metadata, error)
}

type FileProvider struct {
	mu             sync.RWMutex
	metatadaLookup map[string]*Metadata
	infoURL        string
	cacheFile      string
	lastUpdate     time.Time
	cacheDuration  time.Duration
}

func New(infoFilePath string) (Provider, error) {
	info := &FileProvider{
		metatadaLookup: make(map[string]*Metadata),
		infoURL:        "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json",
		cacheFile:      infoFilePath,
		cacheDuration:  24 * time.Hour,
	}
	err := info.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load info data: %w", err)
	}

	return info, nil
}

func (p *FileProvider) Clear() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clear the in-memory cache
	p.metatadaLookup = make(map[string]*Metadata)

	// Delete the cached file if it exists
	if p.cacheFile != "" {
		if _, err := os.Stat(p.cacheFile); err == nil {
			if err := os.Remove(p.cacheFile); err != nil {
				return fmt.Errorf("failed to remove info file: %w", err)
			}
		}
	}

	return nil
}

func (p *FileProvider) Load() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If no cache file is specified, just download fresh data
	if p.cacheFile == "" {
		return p.download()
	}

	// Check if we need to update the cache file
	needsUpdate := true
	if info, err := os.Stat(p.cacheFile); err == nil {
		p.lastUpdate = info.ModTime()
		if time.Since(p.lastUpdate) < p.cacheDuration {
			needsUpdate = false
		}
	}

	if needsUpdate {
		return p.update()
	}

	// Load from cache file
	data, err := os.ReadFile(p.cacheFile)
	if err != nil {
		return fmt.Errorf("failed to read info file: %w", err)
	}

	metadata := make(map[string]*Metadata)
	if err := json.Unmarshal(data, &metadata); err != nil {
		log.Println("failed to parse info data: %w", err)
		// Failed to parse download and save fresh data
		return p.update()
	}

	p.metatadaLookup = metadata

	return nil
}

func (p *FileProvider) update() error {
	// Download and save fresh data
	err := p.download()
	if err != nil {
		return fmt.Errorf("failed to download info data: %w", err)
	}
	err = p.save()
	if err != nil {
		return fmt.Errorf("failed to save cache file: %w", err)
	}
	return nil
}

func (p *FileProvider) download() error {
	resp, err := http.Get(p.infoURL)
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

	// Create a temporary map to hold all JSON data including sample_spec
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawData); err != nil {
		return fmt.Errorf("failed to parse raw info data: %w", err)
	}

	p.metatadaLookup = make(map[string]*Metadata)

	// Process each field, skipping "sample_spec"
	for key, value := range rawData {
		if key == "sample_spec" {
			continue
		}

		var metadta *Metadata
		if err := json.Unmarshal(value, &metadta); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to parse info for model %s: %v\n", key, err)
			continue
		}
		p.metatadaLookup[key] = metadta
	}

	return nil
}

func (p *FileProvider) save() error {
	if p.cacheFile == "" {
		// return fmt.Errorf("no cache file specified")
		return nil
	}

	// Save to cache file
	data, err := json.Marshal(p.metatadaLookup)
	if err != nil {
		return fmt.Errorf("failed to marshal info data: %w", err)
	}

	if err := os.WriteFile(p.cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write info file: %w", err)
	}

	return nil
}

func (p *FileProvider) Get(modelName string) (*Metadata, error) {
	// Try the full model name first
	if data, ok := p.metatadaLookup[modelName]; ok {
		return data, nil
	}

	// Try with just the model name without provider prefix
	parts := strings.Split(modelName, "/")
	if len(parts) > 1 {
		modelName := parts[len(parts)-1]
		if data, ok := p.metatadaLookup[modelName]; ok {
			return data, nil
		}
	}

	return nil, fmt.Errorf("no info data for model: %s", modelName)
}

// Parse parses a model string in the format "provider/model"
func Parse(modelStr string, infoProviders Provider) (*Model, error) {
	if modelStr == "" {
		return nil, fmt.Errorf("empty model string")
	}

	parts := strings.Split(modelStr, "/")

	if len(parts) < 2 {
		// For models without explicit provider prefix, try to get info if providers available
		if infoProviders != nil {
			info, err := infoProviders.Get(modelStr)
			if err == nil {
				return &Model{
					Provider: info.LiteLLMProvider,
					Name:     modelStr,
					Metadata: info,
				}, nil
			}
		}
		// If model not found in info or no providers, try to infer provider from model name
		provider := inferProvider(modelStr)
		if provider != "" {
			return &Model{
				Provider: provider,
				Name:     modelStr,
				Metadata: nil,
			}, nil
		}
		return nil, fmt.Errorf("could not determine provider for model: %s", modelStr)
	}

	provider := parts[0]
	name := strings.Join(parts[1:], "/")

	var info2 *Metadata
	if infoProviders != nil {
		info, err := infoProviders.Get(modelStr)
		if err != nil {
			// info not found but we can try using it
			fmt.Fprint(os.Stderr, err)
		}
		info2 = info
	}
	return &Model{
		Provider: provider,
		Name:     name,
		Metadata: info2,
	}, nil
}

// String returns the string representation of the model
func (m *Model) String() string {
	return fmt.Sprintf("%s/%s", m.Provider, m.Name)
}

// inferProvider attempts to determine the provider based on model name patterns
func inferProvider(modelName string) string {
	modelName = strings.ToLower(modelName)

	switch {
	case strings.HasPrefix(modelName, "claude"):
		return "anthropic"
	case strings.HasPrefix(modelName, "deepseek"):
		return "deepseek"
	case strings.HasPrefix(modelName, "gpt"):
		return "openai"
	case strings.HasPrefix(modelName, "gemini"):
		return "google"
	case strings.HasPrefix(modelName, "mistral"):
		return "mistral"
	case strings.HasPrefix(modelName, "llama"):
		return "meta"
	default:
		return ""
	}
}