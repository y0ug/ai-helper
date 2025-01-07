package io

import (
	"fmt"
	"os"
	"path/filepath"
)

// FindConfigFile looks for ai-helper config file in standard locations
func FindConfigFile(dir string) (string, error) {
	// Try current directory first
	for _, ext := range []string{".yaml", ".json"} {
		path := filepath.Join(dir, "ai-helper"+ext)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Then try XDG_CONFIG_HOME/ai-helper or ~/.config/ai-helper
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	configDir = filepath.Join(configDir, "ai-helper")

	for _, ext := range []string{".yaml", ".json"} {
		path := filepath.Join(configDir, "ai-helper"+ext)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no config file found in %s or %s", dir, configDir)
}

// EnsureDirectory ensures the directory exists, creating it if necessary
func EnsureDirectory(path string) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}
