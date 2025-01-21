package llmagent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y0ug/ai-helper/internal/llmcontext"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

func TestNewConversationManager(t *testing.T) {
	cm := NewConversationManager("test-id")
	assert.NotNil(t, cm)
	assert.Equal(t, "test-id", cm.ID)
	assert.Empty(t, cm.Templates)
	assert.Empty(t, cm.History)
	assert.Empty(t, cm.Variables)
}

func TestAddTemplate(t *testing.T) {
	cm := NewConversationManager("test-id")
	template := &PromptTemplate{
		ID:           "test-template",
		SystemPrompt: "system prompt",
		UserPrompt:   "user prompt",
		RequiredVars: []string{"var1"},
		NextStates:   []string{"next-state"},
		Handlers:     make(map[string]TurnHandler),
	}

	err := cm.AddTemplate(template)
	assert.NoError(t, err)
	assert.Contains(t, cm.Templates, "test-template")
	
	// Test empty ID
	err = cm.AddTemplate(&PromptTemplate{})
	assert.Error(t, err)
}

func TestStartConversation(t *testing.T) {
	cm := NewConversationManager("test-id")
	template := &PromptTemplate{
		ID:           "test-template",
		SystemPrompt: "system prompt",
		UserPrompt:   "user prompt",
	}
	
	err := cm.AddTemplate(template)
	assert.NoError(t, err)

	err = cm.StartConversation("test-template")
	assert.NoError(t, err)
	assert.Equal(t, "test-template", cm.CurrentState)

	// Test non-existent template
	err = cm.StartConversation("non-existent")
	assert.Error(t, err)
}

func TestProcessTurn(t *testing.T) {
	cm := NewConversationManager("test-id")
	template := &PromptTemplate{
		ID:           "test-template",
		SystemPrompt: "system prompt",
		UserPrompt:   "user prompt",
	}
	
	err := cm.AddTemplate(template)
	assert.NoError(t, err)
	
	err = cm.StartConversation("test-template")
	assert.NoError(t, err)

	reqCtx := &llmcontext.RequestContext{}
	turn, err := cm.ProcessTurn(context.Background(), reqCtx)
	
	assert.NoError(t, err)
	assert.NotNil(t, turn)
	assert.Equal(t, "test-template", turn.TemplateID)
	assert.Equal(t, reqCtx, turn.Input)
	assert.NotZero(t, turn.Timestamp)
	assert.Empty(t, turn.State)
	assert.Len(t, cm.History, 1)
}

type mockTurnHandler struct {
	preProcessCalled  bool
	postProcessCalled bool
}

func (m *mockTurnHandler) PreProcess(ctx context.Context, turn *Turn) error {
	m.preProcessCalled = true
	return nil
}

func (m *mockTurnHandler) PostProcess(ctx context.Context, turn *Turn, response []*chat.ChatResponse) error {
	m.postProcessCalled = true
	return nil
}

func TestTurnHandlers(t *testing.T) {
	cm := NewConversationManager("test-id")
	handler := &mockTurnHandler{}
	
	template := &PromptTemplate{
		ID:           "test-template",
		SystemPrompt: "system prompt",
		UserPrompt:   "user prompt",
		Handlers:     map[string]TurnHandler{"test": handler},
	}
	
	err := cm.AddTemplate(template)
	assert.NoError(t, err)
	
	err = cm.StartConversation("test-template")
	assert.NoError(t, err)

	reqCtx := &llmcontext.RequestContext{}
	_, err = cm.ProcessTurn(context.Background(), reqCtx)
	
	assert.NoError(t, err)
	assert.True(t, handler.preProcessCalled)
}
