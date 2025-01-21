package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

//go:embed templates/*.yaml
var promptFS embed.FS

// LoadPrompts loads all prompt configurations from embedded YAML files
func LoadPrompts() (*PromptsConfig, error) {
	config := &PromptsConfig{}
	yamlData := make(map[string]interface{})

	// Read directory entries
	entries, err := promptFS.ReadDir("templates")
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// First pass: load all YAML files and their imports
	err = loadEmbeddedYAMLFiles(entries, yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to load YAML files: %w", err)
	}

	// Merge all configurations
	mergedYAML, err := mergeYAMLData(yamlData)
	if err != nil {
		return nil, fmt.Errorf("failed to merge YAML data: %w", err)
	}

	// Unmarshal merged configuration
	err = yaml.Unmarshal(mergedYAML, config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

func RenderTemplate(tmpl string, data TemplateData) (string, error) {
	t, err := template.New("prompt").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func loadEmbeddedYAMLFiles(entries []fs.DirEntry, yamlData map[string]interface{}) error {
	processedFiles := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		err := loadYAMLFileWithDeps(entry.Name(), yamlData, processedFiles)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func loadYAMLFileWithDeps(
	filename string,
	yamlData map[string]interface{},
	processedFiles map[string]bool,
) error {
	if processedFiles[filename] {
		return nil
	}

	filePath := path.Join("templates", filename)
	data, err := promptFS.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	var config YAMLConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("failed to unmarshal %s: %w", filename, err)
	}

	// Process imports first
	if len(config.Imports) > 0 {
		for _, importFile := range config.Imports {
			err := loadYAMLFileWithDeps(importFile, yamlData, processedFiles)
			if err != nil {
				return fmt.Errorf("failed to load import %s: %w", importFile, err)
			}
		}
	}

	// Store the data
	yamlData[filename] = config.Data
	processedFiles[filename] = true

	return nil
}

func mergeYAMLData(data map[string]interface{}) ([]byte, error) {
	merged := make(map[string]interface{})

	// Merge all configurations
	for _, value := range data {
		if m, ok := value.(map[string]interface{}); ok {
			for k, v := range m {
				merged[k] = v
			}
		}
	}

	// Convert back to YAML
	return yaml.Marshal(merged)
}
