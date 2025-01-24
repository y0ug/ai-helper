package models

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Constants
const (
	DefaultModelName    = "gpt-4o"
	AnthropicBetaHeader = "prompt-caching-2024-07-31,pdfs-2024-09-25"
)

// ModelSettings represents configuration for a model
type ModelSettings struct {
	Name             string                 `json:"name"`
	EditFormat       string                 `json:"edit_format"`
	WeakModelName    *string                `json:"weak_model_name,omitempty"`
	UseRepoMap       bool                   `json:"use_repo_map"`
	SendUndoReply    bool                   `json:"send_undo_reply"`
	Lazy             bool                   `json:"lazy"`
	Reminder         string                 `json:"reminder"`
	ExamplesAsSysMsg bool                   `json:"examples_as_sys_msg"`
	ExtraParams      map[string]interface{} `json:"extra_params,omitempty"`
	CacheControl     bool                   `json:"cache_control"`
	CachesByDefault  bool                   `json:"caches_by_default"`
	UseSystemPrompt  bool                   `json:"use_system_prompt"`
	UseTemperature   bool                   `json:"use_temperature"`
	Streaming        bool                   `json:"streaming"`
	EditorModelName  *string                `json:"editor_model_name,omitempty"`
	EditorEditFormat *string                `json:"editor_edit_format,omitempty"`
}

// Model represents an AI model with its configuration and capabilities
type Model struct {
	ModelSettings
	MaxChatHistoryTokens int
	WeakModel            *Model
	EditorModel          *Model
	Info                 map[string]interface{}
	MissingKeys          []string
	KeysInEnvironment    bool
}

// ModelInfoManager handles caching and retrieval of model information
type ModelInfoManager struct {
	ModelInfoURL string
	CacheTTL     time.Duration
	CacheDir     string
	CacheFile    string
	Content      map[string]interface{}
}

func InitializeDefaultRegistry() *ModelRegistry {
	registry := NewModelRegistry()
	for _, settings := range DefaultModelSettings {
		registry.RegisterModel(settings)
	}
	return registry
}

// NewModelInfoManager creates a new ModelInfoManager instance
func NewModelInfoManager() *ModelInfoManager {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".aider", "caches")

	return &ModelInfoManager{
		ModelInfoURL: "https://raw.githubusercontent.com/BerriAI/litellm/main/model_prices_and_context_window.json",
		CacheTTL:     24 * time.Hour,
		CacheDir:     cacheDir,
		CacheFile:    filepath.Join(cacheDir, "model_prices_and_context_window.json"),
		Content:      make(map[string]interface{}),
	}
}

// GetTokenCountForImage calculates token cost for an image
func (m *Model) GetTokenCountForImage(fname string) (int, error) {
	file, err := os.Open(fname)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, err
	}

	width, height := img.Width, img.Height

	// Scale down if larger than 2048 in any dimension
	maxDimension := math.Max(float64(width), float64(height))
	if maxDimension > 2048 {
		scaleFactor := 2048 / maxDimension
		width = int(float64(width) * scaleFactor)
		height = int(float64(height) * scaleFactor)
	}

	// Scale to minimum 768 pixels
	minDimension := math.Min(float64(width), float64(height))
	scaleFactor := 768 / minDimension
	width = int(float64(width) * scaleFactor)
	height = int(float64(height) * scaleFactor)

	// Calculate tiles
	tilesWidth := int(math.Ceil(float64(width) / 512))
	tilesHeight := int(math.Ceil(float64(height) / 512))
	numTiles := tilesWidth * tilesHeight

	// Calculate token cost
	tokenCost := numTiles*170 + 85
	return tokenCost, nil
}

// NewModel creates a new Model instance
func NewModel(
	modelName string,
	weakModel *Model,
	editorModel *Model,
	editorEditFormat string,
) (*Model, error) {
	// Initialize model with default settings
	model := &Model{
		ModelSettings: ModelSettings{
			Name:            modelName,
			EditFormat:      "whole",
			UseSystemPrompt: true,
			UseTemperature:  true,
			Streaming:       true,
		},
		MaxChatHistoryTokens: 1024,
	}

	// Configure model settings based on name
	if err := model.configureModelSettings(); err != nil {
		return nil, err
	}

	// Set up weak model if specified
	if weakModel != nil {
		model.WeakModel = weakModel
	}

	// Set up editor model if specified
	if editorModel != nil {
		model.EditorModel = editorModel
		if editorEditFormat != "" {
			model.EditorEditFormat = &editorEditFormat
		}
	}

	return model, nil
}

// Additional model-related constants and mappings
var (
	OpenAIModels = []string{
		"gpt-4",
		"gpt-4o",
		"gpt-4o-2024-05-13",
		"gpt-4-turbo-preview",
		"gpt-4-0314",
		"gpt-4-0613",
		"gpt-4-32k",
		"gpt-4-32k-0314",
		"gpt-4-32k-0613",
		"gpt-4-turbo",
		"gpt-4-turbo-2024-04-09",
		"gpt-4-1106-preview",
		"gpt-4-0125-preview",
		"gpt-4-vision-preview",
		"gpt-4o-mini",
		"gpt-4o-mini-2024-07-18",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-0301",
		"gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-1106",
		"gpt-3.5-turbo-0125",
		"gpt-3.5-turbo-16k",
		"gpt-3.5-turbo-16k-0613",
	}

	AnthropicModels = []string{
		"claude-2",
		"claude-2.1",
		"claude-3-haiku-20240307",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-5-sonnet-20240620",
		"claude-3-5-sonnet-20241022",
	}

	ModelAliases = map[string]string{
		"sonnet":   "claude-3-5-sonnet-20241022",
		"haiku":    "claude-3-5-haiku-20241022",
		"opus":     "claude-3-opus-20240229",
		"4":        "gpt-4-0613",
		"4o":       "gpt-4o",
		"4-turbo":  "gpt-4-1106-preview",
		"35turbo":  "gpt-3.5-turbo",
		"35-turbo": "gpt-3.5-turbo",
		"3":        "gpt-3.5-turbo",
		"deepseek": "deepseek/deepseek-chat",
		"r1":       "deepseek/deepseek-reasoner",
		"flash":    "gemini/gemini-2.0-flash-exp",
	}
)

// ModelInfo represents information about a model's capabilities and requirements
type ModelInfo struct {
	MaxInputTokens  int                    `json:"max_input_tokens"`
	LitellmProvider string                 `json:"litellm_provider"`
	ExtraParams     map[string]interface{} `json:"extra_params"`
}

// LoadCache loads the model information from cache
func (m *ModelInfoManager) LoadCache() error {
	// Ensure cache directory exists
	if err := os.MkdirAll(m.CacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Check if cache file exists and is within TTL
	if info, err := os.Stat(m.CacheFile); err == nil {
		if time.Since(info.ModTime()) < m.CacheTTL {
			data, err := os.ReadFile(m.CacheFile)
			if err != nil {
				return fmt.Errorf("failed to read cache file: %w", err)
			}

			if err := json.Unmarshal(data, &m.Content); err != nil {
				return fmt.Errorf("failed to parse cache file: %w", err)
			}
			return nil
		}
	}

	return nil
}

// UpdateCache updates the model information cache
func (m *ModelInfoManager) UpdateCache() error {
	// TODO: Implement HTTP request to fetch latest model information
	// For now, we'll just save an empty cache if nothing exists
	if m.Content == nil {
		m.Content = make(map[string]interface{})
	}

	data, err := json.MarshalIndent(m.Content, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache content: %w", err)
	}

	if err := os.WriteFile(m.CacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// GetModelInfo retrieves information about a specific model
func (m *Model) GetModelInfo() error {
	manager := NewModelInfoManager()
	if err := manager.LoadCache(); err != nil {
		return err
	}

	modelInfo, exists := manager.Content[m.Name]
	if !exists {
		// Try to get info for base model name if it's a provider-specific model
		parts := strings.Split(m.Name, "/")
		if len(parts) == 2 {
			modelInfo, exists = manager.Content[parts[1]]
		}
	}

	if exists {
		m.Info = modelInfo.(map[string]interface{})
	}

	return nil
}

// ValidateEnvironment checks if required environment variables are set
func (m *Model) ValidateEnvironment() error {
	// Fast path for common models
	if missingKeys := m.FastValidateEnvironment(); missingKeys != nil {
		return fmt.Errorf("missing required environment variables: %v", missingKeys)
	}

	// Check provider-specific requirements
	provider := ""
	if m.Info != nil {
		if p, ok := m.Info["litellm_provider"].(string); ok {
			provider = strings.ToLower(p)
		}
	}

	var requiredVars []string
	switch provider {
	case "cohere_chat":
		requiredVars = []string{"COHERE_API_KEY"}
	case "gemini":
		requiredVars = []string{"GEMINI_API_KEY"}
	case "groq":
		requiredVars = []string{"GROQ_API_KEY"}
	}

	missingVars := m.validateVariables(requiredVars)
	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	return nil
}

// FastValidateEnvironment performs quick validation for common models
func (m *Model) FastValidateEnvironment() []string {
	var requiredVar string

	if strings.HasPrefix(m.Name, "openai/") || contains(OpenAIModels, m.Name) {
		requiredVar = "OPENAI_API_KEY"
	} else if strings.HasPrefix(m.Name, "anthropic/") || contains(AnthropicModels, m.Name) {
		requiredVar = "ANTHROPIC_API_KEY"
	} else {
		return nil
	}

	if _, exists := os.LookupEnv(requiredVar); !exists {
		return []string{requiredVar}
	}

	return nil
}

// validateVariables checks if required environment variables are set
func (m *Model) validateVariables(vars []string) []string {
	var missingVars []string
	for _, v := range vars {
		if _, exists := os.LookupEnv(v); !exists {
			missingVars = append(missingVars, v)
		}
	}
	return missingVars
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (m *Model) configureModelSettings() error {
	// First try exact model match from default settings
	exactMatch := false
	for _, ms := range DefaultModelSettings {
		if m.Name == ms.Name {
			m.copyModelSettings(ms)
			exactMatch = true
			break
		}
	}

	// If no exact match found, apply generic settings based on model name
	if !exactMatch {
		m.applyGenericModelSettings()
	}

	// Apply extra params if they exist
	if err := m.applyExtraParams(); err != nil {
		return err
	}

	return nil
}

func (m *Model) copyModelSettings(ms ModelSettings) {
	m.EditFormat = ms.EditFormat
	m.WeakModelName = ms.WeakModelName
	m.UseRepoMap = ms.UseRepoMap
	m.SendUndoReply = ms.SendUndoReply
	m.Lazy = ms.Lazy
	m.Reminder = ms.Reminder
	m.ExamplesAsSysMsg = ms.ExamplesAsSysMsg
	m.ExtraParams = ms.ExtraParams
	m.CacheControl = ms.CacheControl
	m.CachesByDefault = ms.CachesByDefault
	m.UseSystemPrompt = ms.UseSystemPrompt
	m.UseTemperature = ms.UseTemperature
	m.Streaming = ms.Streaming
	m.EditorModelName = ms.EditorModelName
	m.EditorEditFormat = ms.EditorEditFormat
}

// applyGenericModelSettings applies settings based on model name patterns
func (m *Model) applyGenericModelSettings() {
	modelName := strings.ToLower(m.Name)

	switch {
	case strings.Contains(modelName, "llama3") || strings.Contains(modelName, "llama-3"):
		if strings.Contains(modelName, "70b") {
			m.EditFormat = "diff"
			m.UseRepoMap = true
			m.SendUndoReply = true
			m.ExamplesAsSysMsg = true
		}

	case strings.Contains(modelName, "gpt-4-turbo") ||
		(strings.Contains(modelName, "gpt-4-") && strings.Contains(modelName, "-preview")):
		m.EditFormat = "udiff"
		m.UseRepoMap = true
		m.SendUndoReply = true

	case strings.Contains(modelName, "gpt-4") || strings.Contains(modelName, "claude-3-opus"):
		m.EditFormat = "diff"
		m.UseRepoMap = true
		m.SendUndoReply = true

	case strings.Contains(modelName, "gpt-3.5") || strings.Contains(modelName, "gpt-4"):
		m.Reminder = "sys"

	case strings.Contains(modelName, "3.5-sonnet") || strings.Contains(modelName, "3-5-sonnet"):
		m.EditFormat = "diff"
		m.UseRepoMap = true
		m.ExamplesAsSysMsg = true
		m.Reminder = "user"

	case strings.HasPrefix(modelName, "o1-") || strings.Contains(modelName, "/o1-"):
		m.UseSystemPrompt = false
		m.UseTemperature = false

	case strings.Contains(modelName, "qwen") &&
		strings.Contains(modelName, "coder") &&
		(strings.Contains(modelName, "2.5") || strings.Contains(modelName, "2-5")) &&
		strings.Contains(modelName, "32b"):
		m.EditFormat = "diff"
		m.EditorEditFormat = ptrStr("editor-diff")
		m.UseRepoMap = true
		if strings.HasPrefix(modelName, "ollama/") || strings.HasPrefix(modelName, "ollama_chat/") {
			m.ExtraParams = map[string]interface{}{
				"num_ctx": 8 * 1024,
			}
		}
	}

	// If edit_format is "diff", set use_repo_map to true by default
	if m.EditFormat == "diff" {
		m.UseRepoMap = true
	}
}

// applyExtraParams applies any additional parameters from extra model settings
func (m *Model) applyExtraParams() error {
	// Find extra settings
	var extraSettings *ModelSettings
	for _, ms := range DefaultModelSettings {
		if ms.Name == "aider/extra_params" {
			extraSettings = &ms
			break
		}
	}

	if extraSettings != nil && extraSettings.ExtraParams != nil {
		// Initialize extra_params if it doesn't exist
		if m.ExtraParams == nil {
			m.ExtraParams = make(map[string]interface{})
		}

		// Deep merge the extra_params maps
		if err := m.mergeExtraParams(extraSettings.ExtraParams); err != nil {
			return fmt.Errorf("failed to merge extra params: %w", err)
		}
	}

	return nil
}

// mergeExtraParams performs a deep merge of extra parameters
func (m *Model) mergeExtraParams(params map[string]interface{}) error {
	for key, value := range params {
		switch v := value.(type) {
		case map[string]interface{}:
			// If the existing value is also a map, merge recursively
			if existing, ok := m.ExtraParams[key].(map[string]interface{}); ok {
				newMap := make(map[string]interface{})
				for k, v := range existing {
					newMap[k] = v
				}
				for k, v := range v {
					newMap[k] = v
				}
				m.ExtraParams[key] = newMap
			} else {
				m.ExtraParams[key] = v
			}
		default:
			// For non-map values, simply update
			m.ExtraParams[key] = v
		}
	}
	return nil
}
