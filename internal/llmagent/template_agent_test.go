package llmagent

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
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
		Choice: []chat.Choice{
			{
				Content: []*chat.MessageContent{
					chat.NewTextContent("Initial response"),
				},
			},
		},
		Usage: chat.Usage{
			InputTokens:  10,
			OutputTokens: 20,
		},
	}

	mockChat.EXPECT().Stream(
		gomock.Any(),
		gomock.Any(),
	).Return(chat.NewMockEventStream(initialResponse), nil)

	// Load initial input
	err = agent.LoadArgs(map[string]string{"Input": "Hello"})
	assert.NoError(t, err)

	// Execute initial prompt
	resp, cost, err := agent.Execute(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Greater(t, cost, float64(0))

	// Verify conversation state
	assert.Equal(t, "initial", agent.ConversationManager.CurrentState)
	assert.Len(t, agent.ConversationManager.History, 1)

	// Test follow-up message
	followUpResponse := &chat.ChatResponse{
		Choice: []chat.Choice{
			{
				Content: []*chat.MessageContent{
					chat.NewTextContent("Follow-up response"),
				},
			},
		},
		Usage: chat.Usage{
			InputTokens:  15,
			OutputTokens: 25,
		},
	}

	mockChat.EXPECT().Stream(
		gomock.Any(),
		gomock.Any(),
	).Return(chat.NewMockEventStream(followUpResponse), nil)

	// Change state and load follow-up input
	agent.ConversationManager.CurrentState = "follow_up"
	err = agent.LoadArgs(map[string]string{"Input": "Tell me more"})
	assert.NoError(t, err)

	// Execute follow-up prompt
	resp, cost, err = agent.Execute(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Greater(t, cost, float64(0))

	// Verify conversation history
	assert.Len(t, agent.ConversationManager.History, 2)
	assert.Equal(t, "follow_up", agent.ConversationManager.History[1].TemplateID)
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
