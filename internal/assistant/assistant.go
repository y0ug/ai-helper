package assistant

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/actions/executors"
	"github.com/y0ug/ai-helper/internal/assistant/extractors"
	"github.com/y0ug/ai-helper/internal/assistant/llm"
	"github.com/y0ug/ai-helper/internal/assistant/llm/metrics"
	"github.com/y0ug/ai-helper/internal/assistant/llm/models"
	"github.com/y0ug/ai-helper/internal/assistant/prompts"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/settings"
	"github.com/y0ug/ai-helper/internal/assistant/validation"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type AssistantOptions struct {
	MainModel    *models.Model
	RepoManager  repomanager.RepoManagerInterface
	LlmClient    chat.Provider
	Logger       *slog.Logger
	Prompts      prompts.Prompter
	Settings     *settings.CoderSettings
	StreamWriter io.Writer
	Stream       bool
	ConfirmChan  chan actions.Action
	ResponseChan chan executors.UserResponse
}

type AssistantOrchestrator struct {
	logger     *slog.Logger
	llm        llm.ChatCompleter
	prompts    prompts.Prompter
	rm         repomanager.RepoManagerInterface
	settings   *settings.CoderSettings
	processor  *ActionExecutor
	history    *ChatHistory
	formatter  *PromptFormatter
	extractors []extractors.Extractor
	metrics    llm.MetricsRecorder
}

func NewAssistantOrchestrator(opts AssistantOptions) *AssistantOrchestrator {
	history := NewChatHistory()
	formatter := NewPromptFormatter(opts.Logger, opts.RepoManager, opts.Prompts, opts.Settings)
	metricsTracker := metrics.NewMetricsTracker(opts.Logger, *opts.Settings.MainModel())
	c := &AssistantOrchestrator{
		rm:        opts.RepoManager,
		logger:    opts.Logger,
		settings:  opts.Settings,
		history:   history,
		formatter: formatter,
		metrics:   metricsTracker,
	}

	// Stream processor for the LLMClient wrapper
	var streamProcessor *llm.StreamProcessor
	if opts.Stream {
		streamProcessor = llm.NewStreamProcessor(opts.StreamWriter, opts.Logger)
	}

	// Generate the LLMClient wrapper
	c.llm = llm.New(
		opts.LlmClient,
		opts.Settings,
		opts.Logger,
		streamProcessor,
		metricsTracker,
	)

	validator := validation.NewValidationPipeline(c.logger)
	// validator.AddStep(validation.NewDryRunValidator())

	registry := executors.NewRegistryFull(
		c.logger,
		c.rm,
		validator,
		opts.ConfirmChan,
		opts.ResponseChan,
	)

	// Should handle this better
	c.SetPrompts(opts.Prompts)
	processor := NewActionExecutor(
		opts.Logger,
		opts.RepoManager,
		history,
		formatter,
		opts.Settings,
		opts.Prompts,
		c.extractors,
		registry,
	)
	c.processor = processor

	re := make([]string, 0)
	features := make([]string, 0)
	for _, e := range c.extractors {
		re = append(re, e.Name())
		actions := e.SupportedActions()
		for _, action := range actions {
			features = append(features, string(action))
		}
	}

	c.logger.Info(
		"setting ",
		"max_output_token", c.settings.GetMaxOutputToken(),
		"model_name", c.settings.GetModelName(),
		"prompt_name",
		opts.Prompts.GetName(),
		"extractors",
		re,
		"extractor_features",
		features,
	)
	return c
}

func (c *AssistantOrchestrator) SetPrompts(pts prompts.Prompter) {
	c.prompts = pts

	extractorNames := strings.Split(c.getPrompts().GetEditFormat(), "\n")
	for _, name := range extractorNames {
		extractorName := extractors.New(extractors.ExtractorType(name), c.logger)
		if extractorName == nil {
			c.logger.Error("Error creating extractor", "name", extractorName)
			continue
		}
		c.extractors = append(c.extractors, extractorName)
	}

	// c.templateHandler = prompts.NewTemplateHandler(
	// 	c.prompts,
	// 	c.settings,
	// 	c.logger,
	// )
}

func (c *AssistantOrchestrator) GetRM() repomanager.RepoManagerInterface {
	return c.rm
}

func (c *AssistantOrchestrator) getPrompts() prompts.Prompter {
	return c.prompts
}

func (c *AssistantOrchestrator) initBeforeMessage() {
	// Reset state before processing a new message
	// if c.rm.GetGit() != nil {
	// 	// Should commit before message
	// 	lastCommitHash, err := c.rm.GetGit().GetHeadCommitSHA(false)
	// 	if err != nil {
	// 		fmt.Println("Error getting head commit SHA:", err)
	// 	} else {
	// 		c.lastCommitHash = lastCommitHash
	// 	}
	// }
}

func (c *AssistantOrchestrator) Run(ctx context.Context, message string) error {
	c.initBeforeMessage()
	return c.SendMessage(ctx, message)
}

func (c *AssistantOrchestrator) SendMessage(ctx context.Context, message string) error {
	// Add user message
	c.history.AddMessage(chat.NewMessage("user", chat.NewTextContent(message)))

	// Format messages with appropriate prompts
	i := 0
	messages := c.FormatMessages()

	for len(c.history.GetCurrentMessages()) > 0 && i < 4 {
		messages.Cur = c.history.GetCurrentMessages()
		tools := make([]chat.Tool, 0)
		for _, e := range c.extractors {
			tools = append(tools, e.GetChatTools()...)
		}

		for _, tool := range tools {
			c.logger.Debug("SendMessage: tool", "name", tool.Name)
		}
		for _, m := range messages.AllMessages() {
			c.logger.Debug(
				"SendMessage: msg",
				"role",
				m.Role,
				"is_cacheable",
				m.Content[0].IsCacheable(),
				"content",
				m.Content,
			)
		}
		resp, err := c.llm.SendMessages(ctx, messages.AllMessages(), tools)
		if err != nil {
			return err
		}

		c.logger.Info("metrics", "total", c.metrics)

		err = c.processor.ProcessResponse(ctx, resp)
		if err != nil {
			return fmt.Errorf("error processing response: %w", err)
		}
		i += 1
	}

	return nil
}

// FormatMessages formats all messages for the LLM with appropriate prompts
func (c *AssistantOrchestrator) FormatMessages() *PromptChunks {
	chunks := c.formatter.FormatMessages(c.history)

	return chunks
}
