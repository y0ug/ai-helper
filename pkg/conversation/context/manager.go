package context

import (
	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/filemanager"
)

// conversationContext implements ContextManager
type conversationContext struct {
	command *config.Command
	env     map[string]string
	fm      filemanager.FileManager
	vars    map[string]interface{}
}

func NewContext(cmd *config.Command, fm filemanager.FileManager) *conversationContext {
	return &conversationContext{
		command: cmd,
		env:     make(map[string]string),
		fm:      fm,
		vars:    make(map[string]interface{}),
	}
}
