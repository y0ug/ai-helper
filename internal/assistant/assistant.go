package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/actions/executors"
	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
	"github.com/y0ug/ai-helper/internal/assistant/extractors"
	"github.com/y0ug/ai-helper/internal/assistant/llm"
	"github.com/y0ug/ai-helper/internal/assistant/llm/metrics"
	"github.com/y0ug/ai-helper/internal/assistant/llm/models"
	"github.com/y0ug/ai-helper/internal/assistant/prompt"
	"github.com/y0ug/ai-helper/internal/assistant/prompt/prompts"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/settings"
	"github.com/y0ug/ai-helper/internal/assistant/validation"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type AssistantOptions struct {
	MainModel   *models.Model
	RepoManager repomanager.RepoManagerInterface
	LlmClient   chat.Provider
	Logger      *slog.Logger
	Prompts     prompts.Prompter
	Settings    *settings.CoderSettings
	Stream      bool
	eventBus    *eventbus.EventBus
}

type AssistantOrchestrator struct {
	logger       *slog.Logger
	llm          llm.ChatCompleter
	prompts      prompts.Prompter
	rm           repomanager.RepoManagerInterface
	settings     *settings.CoderSettings
	processor    *ActionExecutor
	history      *prompt.ChatHistory
	formatter    *prompt.PromptFormatter
	extractors   []extractors.Extractor
	metrics      llm.MetricsRecorder
	runCtx       context.Context
	cancelRunCtx context.CancelFunc
	eventBus     *eventbus.EventBus
	lastMsgChunk *prompt.PromptChunks
}

func NewAssistantOrchestrator(opts AssistantOptions) *AssistantOrchestrator {
	history := prompt.NewChatHistory()
	formatter := prompt.NewPromptFormatter(
		opts.Logger,
		opts.RepoManager,
		opts.Prompts,
		opts.Settings,
	)
	metricsTracker := metrics.NewMetricsTracker(opts.Logger, *opts.Settings.MainModel())
	c := &AssistantOrchestrator{
		rm:        opts.RepoManager,
		logger:    opts.Logger,
		settings:  opts.Settings,
		history:   history,
		formatter: formatter,
		metrics:   metricsTracker,
		eventBus:  eventbus.GetEventBus(),
	}

	c.registerEventHandlers()

	outputChan := make(chan string)
	go func() {
		for content := range outputChan {
			c.eventBus.Publish(eventbus.NewEvent(eventbus.EventOutput, content))
		}
	}()

	// Stream processor for the LLMClient wrapper
	var streamProcessor *llm.StreamProcessor
	if opts.Stream {
		streamProcessor = llm.NewStreamProcessor(
			outputChan,
			opts.Logger,
		)
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

	// Should handle this better
	// This is loading the correct c.extractors
	// we them to be correctly be set before loading executors.NewRegistry and NewActionExecutor
	c.SetPrompts(opts.Prompts)

	registry := executors.NewRegistryFull(
		c.logger,
		c.rm,
		validator,
		nil,
		history,
		c.extractors,
	)

	processor := NewActionExecutor(
		opts.Logger,
		opts.RepoManager,
		history,
		formatter,
		opts.Settings,
		opts.Prompts,
		c.extractors,
		registry,
		nil,
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

// TODO: Proper ctx handling for cancellation
func (c *AssistantOrchestrator) registerEventHandlers() {
	sub := c.eventBus.Subscribe(100)

	go func() {
		for event := range sub {
			switch event.Type {
			case eventbus.EventInput:
				c.handleInputEvent(event)
			case eventbus.EventLLMRequest:
				c.handleLLMRequestEvent(event)
			case eventbus.EventShutdown:
				c.Stop()
			}
		}
	}()
}

func (c *AssistantOrchestrator) handleLLMRequestEvent(event eventbus.Event) {
	action, ok := event.Payload.(actions.Action)
	if !ok {
		c.logger.Error("Invalid event payload", "type", fmt.Sprintf("%T", event.Payload))
		return
	}
	c.SendMessage(c.runCtx, action)
}

func (c *AssistantOrchestrator) handleInputEvent(event eventbus.Event) {
	input, ok := event.Payload.(eventbus.UserInput)
	if !ok {
		c.logger.Error("Invalid input event payload")
		return
	}

	// c.eventBus.Publish(eventbus.NewEvent(
	// 	eventbus.StatusManager,
	// 	map[string]interface{}{
	// 		"New": "processing",
	// 	}))

	// Process input through existing pipeline
	err := c.Run(c.runCtx, input.Content)
	if err != nil {
		c.eventBus.Publish(eventbus.NewEvent(
			eventbus.EventError,
			map[string]interface{}{
				"error":   err,
				"context": "input_processing",
			},
		))
	}
}

func (a *AssistantOrchestrator) DumpActionChain() {
	a.processor.DumpActionChains()
}

func (a *AssistantOrchestrator) Start(ctx context.Context) {
	a.logger.Info("Starting assistant orchestrator")
	a.runCtx, a.cancelRunCtx = context.WithCancel(ctx)
	// go a.handleAssistantInputChan()
	a.processor.Start(a.runCtx)
}

func (a *AssistantOrchestrator) Stop() {
	a.cancelRunCtx()
	// we shoiuld not have to stop a.processor since the context is cancelled
	a.processor.Stop()
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

// Should be only call on user input
func (c *AssistantOrchestrator) Run(ctx context.Context, message string) error {
	c.initBeforeMessage()

	// Add user message

	c.logger.Info("Run: message", "stop_reason", c.history.GetLastStopReason(), "message", message)
	if c.history.GetLastStopReason() == "end_turn" || c.lastMsgChunk == nil {
		// Rebuilding the last msg chunk if we start a new turn
		c.lastMsgChunk = c.FormatMessages()
		c.history.SetLastStopReason("")
		c.history.MoveCurrentToDone(message)
	} else {
		c.history.AddMessage(chat.NewMessage("user", chat.NewTextContent(message)))
		c.lastMsgChunk.Cur = c.history.GetCurrentMessages()
	}

	// Build current messages to put it in the requests
	promptText := c.lastMsgChunk.ToMarkdown(c.lastMsgChunk.Cur)

	// Create and register a new LLmRequestAction
	llmReqAction := actions.NewLLMRequestAction(promptText)
	// c.processor.actionManager.RegisterAction(llmReqAction)
	c.eventBus.Publish(eventbus.NewEvent(eventbus.EventAction,
		llmReqAction))

	return nil
	// return c.SendMessage(ctx)
}

func (c *AssistantOrchestrator) collectTools() []chat.Tool {
	tools := make([]chat.Tool, 0)
	for _, e := range c.extractors {
		tools = append(tools, e.GetChatTools()...)
	}

	for _, tool := range tools {
		c.logger.Debug("SendMessage: tool", "name", tool.Name)
	}
	return tools
}

func (c *AssistantOrchestrator) SendMessage(ctx context.Context, action actions.Action) error {
	tools := c.collectTools()

	if c.lastMsgChunk == nil {
		c.logger.Warn("lastMsgChunk is nil")
		return fmt.Errorf("lastMsgChunk is nil")
	}
	c.lastMsgChunk.Cur = c.history.GetCurrentMessages()

	resp, err := c.llm.SendMessages(ctx, c.lastMsgChunk.AllMessages(), tools)
	if err != nil {
		return fmt.Errorf("error sending messages: %w", err)
	}

	// Send the response text content to the eventbus output if we are in not stream mode
	// for _, content := range resp.Content {
	// 	if content.Type == chat.ContentTypeText {
	// 	}
	// }

	c.logger.Info("metrics", "total", c.metrics, "resp", resp.ToMessageParams())

	c.eventBus.Publish(eventbus.NewEvent(eventbus.EventAction,
		actions.NewLLMResponseAction(*resp).WithParent(&action)))

	return nil
}

// FormatMessages formats all messages for the LLM with appropriate prompts
func (c *AssistantOrchestrator) FormatMessages() *prompt.PromptChunks {
	chunks := c.formatter.FormatMessages(c.history)

	return chunks
}
