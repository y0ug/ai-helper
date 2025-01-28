package executors

import (
	"context"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
	"github.com/y0ug/ai-helper/internal/assistant/extractors"
	"github.com/y0ug/ai-helper/internal/assistant/prompt"
	"github.com/y0ug/ai-helper/internal/assistant/repomanager"
	"github.com/y0ug/ai-helper/internal/assistant/validation"
)

type Executor interface {
	GetHandler(action actions.Action) ActionHandler
}

type ActionHandler interface {
	CanHandle(actions.Action) bool
	Handle(context.Context, actions.Action) ([]actions.Action, error)
}

type Registry struct {
	handlers []ActionHandler
}

func NewRegistry() *Registry {
	return &Registry{
		handlers: make([]ActionHandler, 0),
	}
}

func (r *Registry) Register(handler ActionHandler) {
	r.handlers = append(r.handlers, handler)
}

func (r *Registry) GetHandler(action actions.Action) ActionHandler {
	for _, handler := range r.handlers {
		if handler.CanHandle(action) {
			return handler
		}
	}
	return nil
}

func NewRegistryFull(
	logger *slog.Logger,
	repo repomanager.RepoManagerInterface,
	validator validation.Validator,
	history *prompt.ChatHistory,
	extractors []extractors.Extractor,
	sendMessage Processor[actions.Action],
	eventBus *eventbus.EventBus,
) *Registry {
	registry := &Registry{
		handlers: make([]ActionHandler, 0),
	}
	registry.Register(NewShellExecutor(
		[]string{`.*`}, // Example safe patterns
		// []string{`^ls$`, `^go test .*`}, // Example safe patterns
		logger,
	))
	registry.Register(NewEditExecutor(repo, validator, logger))
	registry.Register(NewCommitExecutor(repo))
	registry.Register(NewUserInteractionExecutor(logger, eventBus))
	registry.Register(NewLogExecutor())
	registry.Register(NewAddMessageExecutor(history))
	registry.Register(NewExtractorExecutor(logger, extractors))
	registry.Register(NewLLMResponseExecutor(history))
	registry.Register(NewLLMRequestExecutor(sendMessage))
	return registry
}
