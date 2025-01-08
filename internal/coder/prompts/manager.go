package prompts

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/y0ug/ai-helper/internal/prompt"
)

//go:embed templates/*
var templateFS embed.FS

type Manager struct {
	templates map[string]string
}

func NewManager() *Manager {
	return &Manager{
		templates: make(map[string]string),
	}
}

func ResetTemplatesFS() {
	_, _ = template.ParseFS(templateFS, "templates/analyze.tmpl", "templates/init.tmpl")
}

func (m *Manager) LoadTemplates() error {
	entries, err := templateFS.ReadDir("templates")
	if err != nil {
		return fmt.Errorf("failed to read templates directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmpl") {
			content, err := templateFS.ReadFile(filepath.Join("templates", entry.Name()))
			if err != nil {
				return fmt.Errorf("failed to read template %s: %w", entry.Name(), err)
			}

			name := strings.TrimSuffix(entry.Name(), ".tmpl")
			m.templates[name] = string(content)
		}
	}

	return nil
}

func (m *Manager) Execute(templateName string, data *prompt.TemplateData) (string, error) {
	tmpl, exists := m.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}
	return prompt.Execute(tmpl, data)
}
