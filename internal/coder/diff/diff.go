package diff

import (
    "fmt"
    "strings"
)

type Generator struct{}

func NewGenerator() *Generator {
    return &Generator{}
}

// ApplyChanges applies the sections to the provided files
func (g *Generator) ApplyChanges(files map[string]string, sections []Section) (map[string]string, error) {
    result := make(map[string]string)
    for k, v := range files {
        result[k] = v
    }
    
    // Group sections by filename
    fileChanges := make(map[string][]Section)
    for _, section := range sections {
        fileChanges[section.Filename] = append(fileChanges[section.Filename], section)
    }
    
    // Apply changes for each file
    for filename, fileSections := range fileChanges {
        content, exists := result[filename]
        if !exists {
            return nil, fmt.Errorf("file not found: %s", filename)
        }
        
        newContent, err := g.applyFileChanges(content, fileSections)
        if err != nil {
            return nil, fmt.Errorf("failed to apply changes to %s: %w", filename, err)
        }
        
        result[filename] = newContent
    }
    
    return result, nil
}

func (g *Generator) applyFileChanges(content string, sections []Section) (string, error) {
    for _, section := range sections {
        if section.SearchBlock == "" {
            // Handle pure additions
            content += "\n" + section.ReplaceBlock
            continue
        }
        
        if !strings.Contains(content, section.SearchBlock) {
            return "", fmt.Errorf("search block not found: %q", section.SearchBlock)
        }
        
        content = strings.Replace(content, section.SearchBlock, section.ReplaceBlock, 1)
    }
    
    return content, nil
}

// GeneratePatch creates a unified diff for the changes
func (g *Generator) GeneratePatch(original, modified string) string {
    return fmt.Sprintf("--- Original\n+++ Modified\n%s", modified)
}
