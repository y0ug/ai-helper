package assistant

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/actions/executors"
	conversation "github.com/y0ug/ai-helper/internal/assistant/conversation"
	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
	"github.com/y0ug/ai-helper/internal/assistant/extractors"
	"github.com/y0ug/ai-helper/internal/assistant/llm"
	"github.com/y0ug/ai-helper/internal/assistant/llm/metrics"
	"github.com/y0ug/ai-helper/internal/assistant/llm/models"
	"github.com/y0ug/ai-helper/internal/assistant/prompt/prompts"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/settings"
	"github.com/y0ug/ai-helper/internal/assistant/ui"
	"github.com/y0ug/ai-helper/internal/assistant/validation"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

type Assister interface {
	InputEvent(ctx context.Context, event eventbus.Event) error
	AddFileEvent(ctx context.Context, event eventbus.Event) error
	RemoveFileEvent(ctx context.Context, event eventbus.Event) error
}

type AssistantOptions struct {
	MainModel   *models.Model
	RepoManager repomanager.RepoManagerInterface
	LlmClient   chat.Provider
	Logger      *slog.Logger
	Prompts     prompts.Prompter
	Settings    *settings.CoderSettings
	Stream      bool
	EventBus    *eventbus.EventBus
	UI          ui.UserInterface
}

type AssistantOrchestrator struct {
	logger        *slog.Logger
	llm           llm.ChatCompleter
	rm            repomanager.RepoManagerInterface
	settings      *settings.CoderSettings
	processor     *Pipeline
	metrics       llm.MetricsRecorder
	eventBus      *eventbus.EventBus
	actionManager *actions.ActionManager
	status        *ui.StatusManager
	conversation  *conversation.ConversationManager
	llmTools      []chat.Tool
	ui            ui.UserInterface
	mu            sync.Mutex
}

func NewFromPrompts(logger *slog.Logger, pts prompts.Prompter) (results []extractors.Extractor) {
	extractorNames := strings.Split(pts.GetEditFormat(), "\n")
	for _, name := range extractorNames {
		extractorName := extractors.New(extractors.ExtractorType(name), logger)
		if extractorName == nil {
			logger.Error("Error creating extractor", "name", extractorName)
			continue
		}
		results = append(results, extractorName)
	}
	return
}

func NewAssistantOrchestrator(opts AssistantOptions) *AssistantOrchestrator {
	history := conversation.NewChatHistory()

	metricsTracker := metrics.NewMetricsTracker(opts.Logger, *opts.Settings.MainModel())
	c := &AssistantOrchestrator{
		rm:       opts.RepoManager,
		logger:   opts.Logger,
		settings: opts.Settings,
		ui:       opts.UI,
		// history:   history,
		// formatter: formatter,
		metrics:  metricsTracker,
		eventBus: opts.EventBus,
		status:   ui.NewStatusManager(ui.StatusReady),
	}

	c.registerEventHandlers()

	outputChan := make(chan string)
	go func() {
		for content := range outputChan {
			c.ui.Publish(eventbus.NewEvent(eventbus.EventOutput, content))
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

	// ConversationManager
	cm := conversation.NewConversationManager(
		opts.Logger, history, opts.RepoManager, 1024,
		opts.Prompts, opts.Settings)
	c.conversation = cm

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
	exts := NewFromPrompts(c.logger, opts.Prompts)
	c.llmTools = extractors.GetTools(exts...)

	re := make([]string, 0)
	features := make([]string, 0)
	for _, e := range exts {
		re = append(re, e.Name())
		actions := e.SupportedActions()
		for _, action := range actions {
			features = append(features, string(action))
		}
	}

	registry := executors.NewRegistryFull(opts.Logger, c.rm, validator, cm, exts,
		c.SendMessage, c.eventBus, c.ui)
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
				c.HandleInputEvent(ctx, event)
			case eventbus.EventAddFile:
				c.HandleAddFileEvent(ctx, event)
			case eventbus.EventRemoveFile:
				c.HandleRemoveFileEvent(ctx, event)
			case eventbus.EventShutdown:
			}
		}
	}()
}

func (c *AssistantOrchestrator) HandleInputEvent(ctx context.Context, event eventbus.Event) {
	input, ok := event.Payload.(eventbus.UserInput)
	if !ok {
		c.logger.Error("Invalid input event payload")
		return
	}

	// Process input through existing pipeline
	err := c.Run(ctx, input.Content)
	if err != nil {
		c.logger.Error("Error processing input", "error", err)
		c.ui.Publish(eventbus.NewEvent(
			eventbus.EventError,
			map[string]interface{}{
				"error":   err,
				"context": "input_processing",
			},
		))
		c.ui.Error(eventbus.NewEvent(
			eventbus.EventError,
			map[string]interface{}{
				"error":   err,
				"context": "input_processing",
			},
		))
	}
}

func (c *AssistantOrchestrator) HandleAddFileEvent(ctx context.Context, event eventbus.Event) {
	payload, ok := event.Payload.(eventbus.FileOperation)
	if !ok {
		c.logger.Error("Invalid payload for AddFile event")
		return
	}

	c.AddFiles(ctx, payload.ReadOnly, payload.Files...)
}

func (c *AssistantOrchestrator) RemoveFiles(ctx context.Context, filesname ...string) {
	files := make([]string, 0)
	for _, file := range filesname {
		err := c.rm.GetFM().Remove(file)
		if err != nil {
			c.logger.Error("Error adding file", "file", file, "error", err)
			c.ui.Error(eventbus.NewEvent(
				eventbus.EventError,
				map[string]interface{}{
					"error":   err,
					"context": "add_file",
					"file":    file,
				},
			))
		} else {
			files = append(files, file)
		}
	}
	c.ui.Publish(eventbus.NewEvent(
		eventbus.EventFileNotification,
		eventbus.FileOperation{
			Type:  eventbus.FileOperationTypeRemove,
			Files: files,
		}))
}

func (c *AssistantOrchestrator) AddFiles(ctx context.Context, readOnly bool, filesname ...string) {
	files := make([]string, 0)
	for _, file := range filesname {
		err := c.rm.GetFM().Add(file, readOnly)
		if err != nil {
			c.logger.Error("Error adding file", "file", file, "error", err)
			c.ui.Publish(eventbus.NewEvent(
				eventbus.EventError,
				map[string]interface{}{
					"error":   err,
					"context": "add_file",
					"file":    file,
				},
			))
		} else {
			files = append(files, file)
		}
	}
	c.ui.Publish(
		eventbus.NewEvent(
			eventbus.EventFileNotification,
			eventbus.FileOperation{
				Type:     eventbus.FileOperationTypeAdd,
				Files:    files,
				ReadOnly: readOnly,
			}))
}

func (c *AssistantOrchestrator) HandleRemoveFileEvent(ctx context.Context, event eventbus.Event) {
	payload, ok := event.Payload.(eventbus.FileOperation)
	if !ok {
		c.logger.Error("Invalid payload for RemoveFile event")
		return
	}
	c.RemoveFiles(ctx, payload.Files...)
}

func (a *AssistantOrchestrator) DumpActionChain() string {
	var sb strings.Builder

	for _, chain := range a.actionManager.GetAllChains() {
		sb.WriteString(fmt.Sprintf("Action Chain: %s\n", chain.ChainID))

		// Assuming DumpActionChainTree returns a string, if not it needs to be modified
		treeOutput := a.actionManager.DumpActionChainTree(chain.ChainID)
		sb.WriteString(treeOutput)

		sb.WriteString("Execution Timeline:\n")
		sortedActions := chain.GetActionsSorted()
		for i, action := range sortedActions {
			status := "✓"
			if containsError(chain.Results, action.ID) {
				status = "✗"
			}
			sb.WriteString(fmt.Sprintf("%s [%d] %s\n", status, i+1, action.String()))
		}

		sb.WriteString("\nDetailed Results:\n")
		for _, result := range chain.Results {
			sb.WriteString(fmt.Sprintf("- %s\n", result))
		}
		sb.WriteString("--------------------\n")
	}

	return sb.String()
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

func (c *AssistantOrchestrator) GetRM() repomanager.RepoManagerInterface {
	return c.rm
}

func (a *AssistantOrchestrator) Run(ctx context.Context, userInput string) error {
	// 1. Start a new turn
	a.conversation.StartTurn()

	// 2. Add user message
	a.conversation.AddUserMessage(userInput)

	// 3. Maybe summarize older messages if we exceed token usage
	// err := a.conversation.MaybeSummarize(ctx)
	// if err != nil {
	// 	a.logger.Error("Summarizing failed", "error", err)
	// }

	// 4. Build/Update prompt chunks
	promptChunk := a.conversation.BuildPrompt()

	// Build current messages to put it in the requests
	promptText := promptChunk.ToMarkdown("Current", promptChunk.AllMessages())

	fmt.Println(promptChunk.ToMarkdown("Done", promptChunk.Done))
	fmt.Println(promptChunk.ToMarkdown("Current", promptChunk.Cur))

	// Create and register a new LLmRequestAction
	llmReqAction := actions.NewLLMRequestAction(promptText)

	a.processor.Execute(ctx, llmReqAction)
	return nil
}

func (c *AssistantOrchestrator) SendMessage(
	ctx context.Context,
	action actions.Action,
) (results []actions.Action, err error) {
	c.status.Update(ui.StatusProcessing)
	defer c.status.Update(ui.StatusReady)

	promptChunk := c.conversation.BuildPrompt()

	resp, err := c.llm.SendMessages(ctx, promptChunk.AllMessages(), c.llmTools)
	if err != nil {
		return results, fmt.Errorf("error sending messages: %w", err)
	}
	c.logger.Info("metrics", "total", c.metrics, "resp", resp.ToMessageParams())

	results = append(results, actions.NewLLMResponseAction(*resp).WithParent(&action))
	return
}
