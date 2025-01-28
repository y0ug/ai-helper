package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
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
	"github.com/y0ug/ai-helper/internal/assistant/ui"
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
	EventBus    *eventbus.EventBus
}

type AssistantOrchestrator struct {
	logger        *slog.Logger
	llm           llm.ChatCompleter
	prompts       prompts.Prompter
	rm            repomanager.RepoManagerInterface
	settings      *settings.CoderSettings
	processor     *Pipeline
	history       *prompt.ChatHistory
	formatter     *prompt.PromptFormatter
	extractors    []extractors.Extractor
	metrics       llm.MetricsRecorder
	runCtx        context.Context
	cancelRunCtx  context.CancelFunc
	eventBus      *eventbus.EventBus
	lastMsgChunk  *prompt.PromptChunks
	actionManager *actions.ActionManager
	status        *ui.StatusManager
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
		eventBus:  opts.EventBus,
		status:    ui.NewStatusManager("ready"),
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

	re := make([]string, 0)
	features := make([]string, 0)
	for _, e := range c.extractors {
		re = append(re, e.Name())
		actions := e.SupportedActions()
		for _, action := range actions {
			features = append(features, string(action))
		}
	}

	registry := executors.NewRegistryFull(opts.Logger, c.rm, validator, history, c.extractors,
		c.SendMessage)
	c.actionManager = actions.NewActionManager(opts.Logger)
	pipeline := NewPipeline(
		opts.Logger,
		c.actionManager,
		registry,
	)

	c.processor = pipeline
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

func (c *AssistantOrchestrator) GetStatus() *ui.StatusManager {
	return c.status
}

func (c *AssistantOrchestrator) registerEventHandlers() {
	sub := c.eventBus.Subscribe(100)

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		for event := range sub {
			switch event.Type {
			case eventbus.EventInput:
				c.handleInputEvent(ctx, event)
			case eventbus.EventShutdown:
			}
		}
	}()
}

func (c *AssistantOrchestrator) handleInputEvent(ctx context.Context, event eventbus.Event) {
	input, ok := event.Payload.(eventbus.UserInput)
	if !ok {
		c.logger.Error("Invalid input event payload")
		return
	}

	// Process input through existing pipeline
	err := c.Run(ctx, input.Content)
	if err != nil {
		c.logger.Error("Error processing input", "error", err)
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
	for _, chain := range a.actionManager.GetAllChains() {
		fmt.Printf("Action Chain: %s\n", chain.ChainID)
		a.actionManager.DumpActionChainTree(chain.ChainID)

		fmt.Println("Execution Timeline:")
		sortedActions := chain.GetActionsSorted()
		for i, action := range sortedActions {
			status := "✓"
			if containsError(chain.Results, action.ID) {
				status = "✗"
			}
			fmt.Printf("%s [%d] %s\n", status, i+1, action.String())
		}

		fmt.Println("\nDetailed Results:")
		for _, result := range chain.Results {
			fmt.Println("-", result)
		}
		fmt.Println("--------------------")
	}
}

func containsError(results []string, actionID uuid.UUID) bool {
	for _, result := range results {
		if strings.Contains(result, actionID.String()) &&
			(strings.Contains(result, "ERROR") || strings.Contains(result, "failed")) {
			return true
		}
	}
	return false
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

func (c *AssistantOrchestrator) Run(ctx context.Context, message string) error {
	// Add user message

	c.logger.Info("Run: message", "stop_reason", c.history.GetLastStopReason(), "message", message)
	if c.history.GetLastStopReason() == "end_turn" || c.lastMsgChunk == nil {
		if c.lastMsgChunk == nil {
			c.history.AddMessage(chat.NewMessage("user", chat.NewTextContent(message)))
			c.lastMsgChunk = c.FormatMessages()
		} else {
			c.history.SetLastStopReason("")
			c.history.MoveCurrentToDone(message)
		}
	} else {
		c.history.AddMessage(chat.NewMessage("user", chat.NewTextContent(message)))
		c.lastMsgChunk.Cur = c.history.GetCurrentMessages()
	}

	// Build current messages to put it in the requests
	promptText := c.lastMsgChunk.ToMarkdown(c.lastMsgChunk.Cur)

	// Create and register a new LLmRequestAction
	llmReqAction := actions.NewLLMRequestAction(promptText)

	c.processor.Execute(ctx, llmReqAction)
	return nil
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

func (c *AssistantOrchestrator) SendMessage(
	ctx context.Context,
	action actions.Action,
) (results []actions.Action, err error) {
	c.status.Update(ui.StatusProcessing)
	defer c.status.Update(ui.StatusReady)

	tools := c.collectTools()
	if c.lastMsgChunk == nil {
		c.logger.Warn("lastMsgChunk is nil")
		return results, fmt.Errorf("lastMsgChunk is nil")
	}
	c.lastMsgChunk.Cur = c.history.GetCurrentMessages()

	c.logger.Info("SendMessage: request", "test", c.lastMsgChunk.ToMarkdown(c.lastMsgChunk.Cur))
	resp, err := c.llm.SendMessages(ctx, c.lastMsgChunk.AllMessages(), tools)
	if err != nil {
		return results, fmt.Errorf("error sending messages: %w", err)
	}
	c.logger.Info("metrics", "total", c.metrics, "resp", resp.ToMessageParams())

	results = append(results, actions.NewLLMResponseAction(*resp).WithParent(&action))
	return
}

// FormatMessages formats all messages for the LLM with appropriate prompts
func (c *AssistantOrchestrator) FormatMessages() *prompt.PromptChunks {
	chunks := c.formatter.FormatMessages(c.history)

	return chunks
}
