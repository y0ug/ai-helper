package llmagent

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/streaming"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
	"go.uber.org/mock/gomock"
)

func TestNewTemplateAgent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cmd := &config.Command{
		Templates: map[string]config.Template{
			"initial": {
				System: "test system prompt",
				Prompt: "test user prompt",
			},
		},
		InitialState: "initial",
	}

	chatParams := chat.NewChatParams()
	mockProvider := modelinfo.NewMockProvider(ctrl)
	mockProvider.EXPECT().Get(gomock.Any()).Return(&modelinfo.Metadata{
		MaxTokens:          2048,
		InputCostPerToken:  0.001,
		OutputCostPerToken: 0.002,
	}, nil).AnyTimes()

	agent, err := NewTemplateAgent(
		"test-id",
		logger,
		cmd,
		chatParams,
		mockProvider,
		nil,
	)

	assert.NoError(t, err)
	assert.NotNil(t, agent)
	assert.NotNil(t, agent.ConversationManager)
	assert.Equal(t, "initial", agent.ConversationManager.CurrentState)

	// Verify template conversion
	template := agent.ConversationManager.GetCurrentTemplate()
	assert.NotNil(t, template)
	assert.Equal(t, "initial", template.ID)
	assert.Equal(t, "test system prompt", template.SystemPrompt)
	assert.Equal(t, "test user prompt", template.UserPrompt)
}

func TestTemplateAgentIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create a command with multiple templates for testing conversation flow
	cmd := &config.Command{
		Templates: map[string]config.Template{
			"initial": {
				System: "You are a helpful assistant.",
				Prompt: "Initial question: {{ .Vars.Input }}",
				Variables: []config.Variable{
					{Name: "Input", Type: "arg"},
				},
				NextStates: []string{"follow_up"},
			},
			"follow_up": {
				System: "Continue being helpful.",
				Prompt: "Follow-up: {{ .Vars.Input }}",
				Variables: []config.Variable{
					{Name: "Input", Type: "arg"},
				},
				NextStates: []string{"initial"},
			},
		},
		InitialState: "initial",
	}

	chatParams := chat.NewChatParams()
	mockProvider := modelinfo.NewMockProvider(ctrl)
	mockProvider.EXPECT().Get(gomock.Any()).Return(&modelinfo.Metadata{
		MaxTokens:          2048,
		InputCostPerToken:  0.001,
		OutputCostPerToken: 0.002,
	}, nil).AnyTimes()

	// Create mock chat provider
	mockChat := chat.NewMockProvider(ctrl)

	// Create agent
	agent, err := NewTemplateAgent(
		"test-id",
		logger,
		cmd,
		chatParams,
		mockProvider,
		nil,
	)
	assert.NoError(t, err)
	agent.Client = mockChat

	// Test initial message
	initialResponse := &chat.ChatResponse{
		Choice: []chat.ChatChoice{
			{
				Content: []*chat.MessageContent{
					chat.NewTextContent("Initial response"),
				},
				Role: "assistant",
			},
		},
		Usage: &chat.ChatUsage{
			InputTokens:  10,
			OutputTokens: 20,
		},
		Model: "test-model",
	}

	mockStream := streaming.NewMockStreamer[chat.EventStream](ctrl)
	gomock.InOrder(
		mockStream.EXPECT().Next().Return(true),
		mockStream.EXPECT().Current().Return(chat.EventStream{
			Type:  "text_delta",
			Delta: "Initial response",
		}),
		mockStream.EXPECT().Next().Return(true),
		mockStream.EXPECT().Current().Return(chat.EventStream{
			Type:    "message_stop",
			Message: initialResponse,
		}),
		mockStream.EXPECT().Next().Return(false),
		mockStream.EXPECT().Err().Return(nil),
	)

	mockChat.EXPECT().Stream(
		gomock.Any(),
		gomock.Any(),
	).Return(mockStream, nil)

	// Load initial input
	err = agent.LoadArgs(map[string]string{"Input": "Hello"})
	assert.NoError(t, err)

	// Execute initial prompt
	resp, _, err := agent.Execute(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)
	// assert.Greater(t, cost, float64(0))

	// Verify conversation state and history
	assert.Equal(t, "initial", agent.ConversationManager.CurrentState)
	assert.Len(t, agent.ConversationManager.History, 1)
	assert.NotEmpty(t, agent.GetMessages())

	// Test follow-up message
	followUpResponse := &chat.ChatResponse{
		Choice: []chat.ChatChoice{
			{
				Content: []*chat.MessageContent{
					chat.NewTextContent("Follow-up response"),
				},
				Role: "assistant",
			},
		},
		Usage: &chat.ChatUsage{
			InputTokens:  15,
			OutputTokens: 25,
		},
		Model: "test-model",
	}

	mockStream = streaming.NewMockStreamer[chat.EventStream](ctrl)
	gomock.InOrder(
		mockStream.EXPECT().Next().Return(true),
		mockStream.EXPECT().Current().Return(chat.EventStream{
			Type:  "text_delta",
			Delta: "Follow-up response",
		}),
		mockStream.EXPECT().Next().Return(true),
		mockStream.EXPECT().Current().Return(chat.EventStream{
			Type:    "message_stop",
			Message: followUpResponse,
		}),
		mockStream.EXPECT().Next().Return(false),
		mockStream.EXPECT().Err().Return(nil),
		// mockStream.EXPECT().Close().Return(nil).Times(1),
	)

	mockChat.EXPECT().Stream(
		gomock.Any(),
		gomock.Any(),
	).Return(mockStream, nil)

	// Change state and load follow-up input
	agent.ConversationManager.CurrentState = "follow_up"
	err = agent.LoadArgs(map[string]string{"Input": "Tell me more"})
	assert.NoError(t, err)

	// Execute follow-up prompt
	resp, _, err = agent.Execute(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp, 1)
	// assert.Greater(t, cost, float64(0))

	// Verify conversation history
	assert.Len(t, agent.ConversationManager.History, 2)
	assert.Equal(t, "follow_up", agent.ConversationManager.History[1].TemplateID)
	assert.NotEmpty(t, agent.GetMessages())
}

func TestTemplateAgentStateTransitions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Create a command with state transition testing
	cmd := &config.Command{
		Templates: map[string]config.Template{
			"initial": {
				System:     "Initial system prompt",
				Prompt:     "Initial prompt {{ .Vars.Input }}",
				NextStates: []string{"follow_up", "clarification"},
				Variables: []config.Variable{
					{Name: "Input", Type: "arg"},
				},
			},
			"follow_up": {
				System:     "Follow-up system prompt",
				Prompt:     "Follow-up prompt {{ .Vars.Input }}",
				NextStates: []string{"conclusion", "clarification"},
				Variables: []config.Variable{
					{Name: "Input", Type: "arg"},
				},
			},
			"clarification": {
				System:     "Clarification system prompt",
				Prompt:     "Please clarify: {{ .Vars.Input }}",
				NextStates: []string{"follow_up"},
				Variables: []config.Variable{
					{Name: "Input", Type: "arg"},
				},
			},
			"conclusion": {
				System:     "Conclusion system prompt",
				Prompt:     "Concluding: {{ .Vars.Input }}",
				NextStates: []string{"initial"},
				Variables: []config.Variable{
					{Name: "Input", Type: "arg"},
				},
			},
		},
		InitialState: "initial",
	}

	chatParams := chat.NewChatParams()
	mockProvider := modelinfo.NewMockProvider(ctrl)
	mockProvider.EXPECT().Get(gomock.Any()).Return(&modelinfo.Metadata{
		MaxTokens:          2048,
		InputCostPerToken:  0.001,
		OutputCostPerToken: 0.002,
	}, nil).AnyTimes()

	agent, err := NewTemplateAgent(
		"test-id",
		logger,
		cmd,
		chatParams,
		mockProvider,
		nil,
	)
	assert.NoError(t, err)

	// Test explicit state transition
	mockChat := chat.NewMockProvider(ctrl)
	agent.Client = mockChat

	// Setup mock response with state transition command
	transitionResponse := &chat.ChatResponse{
		Choice: []chat.ChatChoice{
			{
				Content: []*chat.MessageContent{
					chat.NewTextContent("!state follow_up"),
				},
				Role: "assistant",
			},
		},
		Usage: &chat.ChatUsage{
			InputTokens:  10,
			OutputTokens: 20,
		},
	}

	mockStream := streaming.NewMockStreamer[chat.EventStream](ctrl)
	gomock.InOrder(
		mockStream.EXPECT().Next().Return(true),
		mockStream.EXPECT().Current().Return(chat.EventStream{
			Type:  "text_delta",
			Delta: "!state follow_up",
		}),
		mockStream.EXPECT().Next().Return(true),
		mockStream.EXPECT().Current().Return(chat.EventStream{
			Type:    "message_stop",
			Message: transitionResponse,
		}),
		mockStream.EXPECT().Next().Return(false),
		mockStream.EXPECT().Err().Return(nil),
	)

	mockChat.EXPECT().Stream(
		gomock.Any(),
		gomock.Any(),
	).Return(mockStream, nil)

	// Execute with initial input
	err = agent.LoadArgs(map[string]string{"Input": "Test input"})
	assert.NoError(t, err)

	resp, _, err := agent.Execute(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Verify state transition
	assert.Equal(t, "follow_up", agent.ConversationManager.CurrentState)

	// Test invalid state transition
	invalidTransitionResponse := &chat.ChatResponse{
		Choice: []chat.ChatChoice{
			{
				Content: []*chat.MessageContent{
					chat.NewTextContent("!state invalid_state"),
				},
				Role: "assistant",
			},
		},
		Usage: &chat.ChatUsage{
			InputTokens:  10,
			OutputTokens: 20,
		},
	}

	mockStream = streaming.NewMockStreamer[chat.EventStream](ctrl)
	gomock.InOrder(
		mockStream.EXPECT().Next().Return(true),
		mockStream.EXPECT().Current().Return(chat.EventStream{
			Type:  "text_delta",
			Delta: "!state invalid_state",
		}),
		mockStream.EXPECT().Next().Return(true),
		mockStream.EXPECT().Current().Return(chat.EventStream{
			Type:    "message_stop",
			Message: invalidTransitionResponse,
		}),
		mockStream.EXPECT().Next().Return(false),
		mockStream.EXPECT().Err().Return(nil),
	)

	mockChat.EXPECT().Stream(
		gomock.Any(),
		gomock.Any(),
	).Return(mockStream, nil)

	// Execute with follow-up input
	err = agent.LoadArgs(map[string]string{"Input": "Another test"})
	assert.NoError(t, err)

	resp, _, err = agent.Execute(context.Background(), nil)
	assert.NoError(t, err)
	
	// State should remain unchanged due to invalid transition
	assert.Equal(t, "follow_up", agent.ConversationManager.CurrentState)
}

func TestTemplateAgentWithMultipleTemplates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cmd := &config.Command{
		Templates: map[string]config.Template{
			"initial": {
				System:     "initial system",
				Prompt:     "initial prompt",
				NextStates: []string{"follow_up"},
			},
			"follow_up": {
				System:     "follow up system",
				Prompt:     "follow up prompt",
				NextStates: []string{"initial"},
			},
		},
		InitialState: "initial",
	}

	chatParams := chat.NewChatParams()
	mockProvider := modelinfo.NewMockProvider(ctrl)
	mockProvider.EXPECT().Get(gomock.Any()).Return(&modelinfo.Metadata{
		MaxTokens:          2048,
		InputCostPerToken:  0.001,
		OutputCostPerToken: 0.002,
	}, nil).AnyTimes()

	agent, err := NewTemplateAgent(
		"test-id",
		logger,
		cmd,
		chatParams,
		mockProvider,
		nil,
	)
	if err != nil {
		t.Errorf("Error creating agent: %v", err)
	}
	assert.NoError(t, err)
	assert.NotNil(t, agent)

	// Verify all templates were added
	assert.Len(t, agent.ConversationManager.Templates, 2)
	assert.Contains(t, agent.ConversationManager.Templates, "initial")
	assert.Contains(t, agent.ConversationManager.Templates, "follow_up")

	// Verify template states
	initialTemplate := agent.ConversationManager.Templates["initial"]
	assert.Contains(t, initialTemplate.NextStates, "follow_up")

	followUpTemplate := agent.ConversationManager.Templates["follow_up"]
	assert.Contains(t, followUpTemplate.NextStates, "initial")
}
