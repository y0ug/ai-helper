package io

import (
	"os"
	"path/filepath"
)

// GetCacheDir returns the XDG cache directory for ai-helper
func GetCacheDir() string {
	cacheHome := os.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			// Fallback to current directory if we can't get home
			return ".cache"
		}
		cacheHome = filepath.Join(homeDir, ".cache")
	}
	return filepath.Join(cacheHome, "ai-helper")
}
