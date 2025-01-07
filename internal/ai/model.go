package ai

import (
	"fmt"
	"strings"
)

// Model represents an AI model configuration
type Model struct {
	Provider string
	Name     string
}

// ParseModel parses a model string in the format "provider/model"
func ParseModel(modelStr string) (*Model, error) {
	parts := strings.Split(modelStr, "/")
	
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid model format: %s, expected format: provider/model", modelStr)
	}

	provider := parts[0]
	name := strings.Join(parts[1:], "/")

	return &Model{
		Provider: provider,
		Name:     name,
	}, nil
}

// String returns the string representation of the model
func (m *Model) String() string {
	return fmt.Sprintf("%s/%s", m.Provider, m.Name)
}
