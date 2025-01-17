package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader handles configuration file loading
type Loader struct{}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{}
}

// Load reads and parses the configuration file
func (l *Loader) Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	return l.LoadData(data, ext[1:])
}

func (l *Loader) LoadData(data []byte, dataType string) (*Config, error) {
	var config Config
	var err error
	switch dataType {
	case "yaml", "yml":
		err = yaml.Unmarshal(data, &config)
	case "json":
		err = json.Unmarshal(data, &config)
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", dataType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return &config, nil
}
