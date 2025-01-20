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
		modelInfoProvider,
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

func TestTemplateAgentWithMultipleTemplates(t *testing.T) {
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
		modelInfoProvider,
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
