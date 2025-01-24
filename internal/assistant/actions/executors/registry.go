package executors

import (
	"context"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
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
