package models

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"gopkg.in/yaml.v3"
)

// TokenCounter handles token counting for different models
type TokenCounter interface {
	CountTokens(text string) (int, error)
	CountMessageTokens(messages []Message) (int, error)
}

// Message represents a chat message
type Message struct {
	Role    string                 `json:"role"`
	Content string                 `json:"content"`
	Name    string                 `json:"name,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// GetRepoMapTokens calculates the token limit for repository mapping
func (m *Model) GetRepoMapTokens() int {
	mapTokens := 1024
	if m.Info != nil {
		if maxInputTokens, ok := m.Info["max_input_tokens"].(float64); ok {
			mapTokens = int(maxInputTokens) / 8
			mapTokens = min(mapTokens, 4096)
			mapTokens = max(mapTokens, 1024)
		}
	}
	return mapTokens
}

// FuzzyModelMatcher handles fuzzy matching of model names
type FuzzyModelMatcher struct {
	models []string
}

// NewFuzzyModelMatcher creates a new FuzzyModelMatcher instance
func NewFuzzyModelMatcher() *FuzzyModelMatcher {
	// Combine all available models
	allModels := make([]string, 0)
	allModels = append(allModels, OpenAIModels...)
	allModels = append(allModels, AnthropicModels...)

	// Add provider-prefixed versions
	var prefixedModels []string
	for _, model := range allModels {
		if strings.Contains(model, "/") {
			prefixedModels = append(prefixedModels, model)
		} else {
			prefixedModels = append(prefixedModels, "openai/"+model)
			prefixedModels = append(prefixedModels, "anthropic/"+model)
		}
	}

	allModels = append(allModels, prefixedModels...)

	return &FuzzyModelMatcher{
		models: allModels,
	}
}

// FindMatches finds models matching the given search term
func (fm *FuzzyModelMatcher) FindMatches(search string) []string {
	search = strings.ToLower(search)

	// First try exact matches
	var exactMatches []string
	for _, model := range fm.models {
		if strings.ToLower(model) == search {
			exactMatches = append(exactMatches, model)
		}
	}
	if len(exactMatches) > 0 {
		return exactMatches
	}

	// Then try contains
	var containsMatches []string
	for _, model := range fm.models {
		if strings.Contains(strings.ToLower(model), search) {
			containsMatches = append(containsMatches, model)
		}
	}
	if len(containsMatches) > 0 {
		return containsMatches
	}

	// Finally try fuzzy matching
	matches := fuzzy.Find(search, fm.models)
	sort.Strings(matches)
	return matches
}

// YAMLConfig handles YAML configuration for models
type YAMLConfig struct {
	ModelsConfig []ModelSettings `yaml:"models"`
}

// LoadModelSettings loads model settings from YAML files
func LoadModelSettings(filePaths []string) ([]ModelSettings, error) {
	var allSettings []ModelSettings

	for _, path := range filePaths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
		}

		var config YAMLConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML from %s: %w", path, err)
		}

		allSettings = append(allSettings, config.ModelsConfig...)
	}

	return allSettings, nil
}

// SaveModelSettings saves model settings to a YAML file
func SaveModelSettings(settings []ModelSettings, filePath string) error {
	config := YAMLConfig{
		ModelsConfig: settings,
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal settings to YAML: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings to file: %w", err)
	}

	return nil
}

// ModelRegistry manages model registration and lookup
type ModelRegistry struct {
	settings map[string]ModelSettings
}

// NewModelRegistry creates a new ModelRegistry instance
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		settings: make(map[string]ModelSettings),
	}
}

// RegisterModel registers a model with its settings
func (r *ModelRegistry) RegisterModel(settings ModelSettings) {
	r.settings[settings.Name] = settings
}

// GetModelSettings retrieves settings for a specific model
func (r *ModelRegistry) GetModelSettings(modelName string) (ModelSettings, bool) {
	// Check for aliases first
	if alias, exists := ModelAliases[modelName]; exists {
		modelName = alias
	}

	settings, exists := r.settings[modelName]
	return settings, exists
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SanityCheckModel performs validation checks on a model
func (m *Model) SanityCheckModel() []string {
	var warnings []string

	if len(m.MissingKeys) > 0 {
		warnings = append(
			warnings,
			fmt.Sprintf("Model %s is missing required environment variables: %v",
				m.Name, m.MissingKeys),
		)
	}

	if !m.KeysInEnvironment {
		warnings = append(
			warnings,
			fmt.Sprintf("Unknown which environment variables are required for model %s",
				m.Name),
		)
	}

	if m.Info == nil {
		warnings = append(
			warnings,
			fmt.Sprintf("Unknown context window size and costs for model %s",
				m.Name),
		)

		matcher := NewFuzzyModelMatcher()
		matches := matcher.FindMatches(m.Name)
		if len(matches) > 0 {
			warnings = append(warnings, "Did you mean one of these?")
			for _, match := range matches {
				warnings = append(warnings, fmt.Sprintf("- %s", match))
			}
		}
	}

	return warnings
}
