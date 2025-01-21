package prompts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLoadPrompts(t *testing.T) {
	t.Run("successfully loads all prompts", func(t *testing.T) {
		config, err := LoadPrompts()
		require.NoError(t, err)
		require.NotNil(t, config)

		// Test base prompts
		assert.NotNil(t, config.BasePrompts)
		assert.NotEmpty(t, config.BasePrompts.LazyPrompt)
		assert.NotEmpty(t, config.BasePrompts.FilesContentGPTEdits)
		assert.NotEmpty(t, config.BasePrompts.FilesContentPrefix)

		// Test architect prompts
		assert.NotNil(t, config.ArchitectPrompts)
		assert.NotEmpty(t, config.ArchitectPrompts.MainSystem)
		assert.Contains(
			t,
			config.ArchitectPrompts.MainSystem,
			"Act as an expert architect engineer",
		)

		// Test ask prompts
		assert.NotNil(t, config.AskPrompts)
		assert.NotEmpty(t, config.AskPrompts.MainSystem)
		assert.Contains(t, config.AskPrompts.MainSystem, "Act as an expert code analyst")

		// Test edit block prompts
		assert.NotNil(t, config.EditBlockPrompts)
		assert.NotEmpty(t, config.EditBlockPrompts.MainSystem)
		assert.Contains(
			t,
			config.EditBlockPrompts.MainSystem,
			"Act as an expert software developer",
		)

		// Test inheritance of base prompts
		assert.Equal(t, config.BasePrompts.LazyPrompt, config.ArchitectPrompts.LazyPrompt)
		assert.Equal(
			t,
			config.BasePrompts.FilesContentPrefix,
			config.EditBlockPrompts.FilesContentPrefix,
		)
	})

	t.Run("template rendering", func(t *testing.T) {
		config, err := LoadPrompts()
		require.NoError(t, err)

		data := TemplateData{
			Language:       "English",
			Hash:           "abc123",
			Message:        "Test commit",
			Platform:       "linux",
			LazyPrompt:     "Test lazy prompt",
			ShellCmdPrompt: "Test shell prompt",
			Fence0:         "```",
			Fence1:         "```",
		}

		// Test architect prompts template
		rendered, err := RenderTemplate(config.ArchitectPrompts.MainSystem, data)
		require.NoError(t, err)
		assert.Contains(t, rendered, "English")
		assert.NotContains(t, rendered, "{{.Language}}")

		// Test files content GPT edits template
		rendered, err = RenderTemplate(config.BasePrompts.FilesContentGPTEdits, data)
		require.NoError(t, err)
		assert.Contains(t, rendered, "abc123")
		assert.Contains(t, rendered, "Test commit")
		assert.NotContains(t, rendered, "{{.Hash}}")
		assert.NotContains(t, rendered, "{{.Message}}")
	})

	t.Run("example messages", func(t *testing.T) {
		config, err := LoadPrompts()
		require.NoError(t, err)

		// Test EditBlockPrompts example messages
		assert.NotEmpty(t, config.EditBlockPrompts.ExampleMessages)
		for _, msg := range config.EditBlockPrompts.ExampleMessages {
			assert.NotEmpty(t, msg.Role)
			assert.NotEmpty(t, msg.Content)
			assert.True(t, msg.Role == "user" || msg.Role == "assistant")
		}

		// Test WholeFilePrompts example messages
		assert.NotEmpty(t, config.WholeFilePrompts.ExampleMessages)
		for _, msg := range config.WholeFilePrompts.ExampleMessages {
			assert.NotEmpty(t, msg.Role)
			assert.NotEmpty(t, msg.Content)
			assert.True(t, msg.Role == "user" || msg.Role == "assistant")
		}
	})

	t.Run("embedded file system access", func(t *testing.T) {
		// Test reading directory
		entries, err := promptFS.ReadDir("templates")
		require.NoError(t, err)
		assert.NotEmpty(t, entries)

		// Check for essential files
		foundFiles := make(map[string]bool)
		for _, entry := range entries {
			foundFiles[entry.Name()] = true
		}

		essentialFiles := []string{
			"base_prompts.yaml",
			"architect_prompts.yaml",
			"ask_prompts.yaml",
			"editblock_prompts.yaml",
		}

		for _, file := range essentialFiles {
			assert.True(t, foundFiles[file], "Missing essential file: "+file)
		}

		// Test reading file content
		content, err := promptFS.ReadFile("templates/base_prompts.yaml")
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		assert.True(t, strings.Contains(string(content), "base_prompts"))
	})

	t.Run("error handling", func(t *testing.T) {
		// Test template rendering with invalid template
		_, err := RenderTemplate("{{.InvalidField}}", TemplateData{})
		assert.Error(t, err)

		// Test template rendering with nil data
		_, err = RenderTemplate("{{.Language}}", TemplateData{})
		assert.Error(t, err)
	})
}

func TestMergeYAMLData(t *testing.T) {
	t.Run("successfully merges YAML data", func(t *testing.T) {
		data := map[string]interface{}{
			"file1.yaml": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
			"file2.yaml": map[string]interface{}{
				"key3": "value3",
				"key4": "value4",
			},
		}

		merged, err := mergeYAMLData(data)
		require.NoError(t, err)
		assert.NotEmpty(t, merged)

		// Unmarshal merged data to verify content
		var result map[string]interface{}
		err = yaml.Unmarshal(merged, &result)
		require.NoError(t, err)

		assert.Equal(t, "value1", result["key1"])
		assert.Equal(t, "value2", result["key2"])
		assert.Equal(t, "value3", result["key3"])
		assert.Equal(t, "value4", result["key4"])
	})

	t.Run("handles empty data", func(t *testing.T) {
		merged, err := mergeYAMLData(map[string]interface{}{})
		require.NoError(t, err)
		assert.NotEmpty(t, merged)

		var result map[string]interface{}
		err = yaml.Unmarshal(merged, &result)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestLoadYAMLFileWithDeps(t *testing.T) {
	t.Run("handles circular dependencies", func(t *testing.T) {
		yamlData := make(map[string]interface{})
		processedFiles := make(map[string]bool)

		err := loadYAMLFileWithDeps("base_prompts.yaml", yamlData, processedFiles)
		require.NoError(t, err)
		assert.True(t, processedFiles["base_prompts.yaml"])
	})

	t.Run("handles non-existent file", func(t *testing.T) {
		yamlData := make(map[string]interface{})
		processedFiles := make(map[string]bool)

		err := loadYAMLFileWithDeps("non_existent.yaml", yamlData, processedFiles)
		assert.Error(t, err)
	})
}
