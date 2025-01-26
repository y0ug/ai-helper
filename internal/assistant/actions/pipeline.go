package actions

import (
	"context"

	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
)

type Processor interface {
	ProcessAction(context.Context, Action) error
}
type Pipeline struct {
	eventBus  *eventbus.EventBus
	processor Processor
}

func NewPipeline(processor Processor) *Pipeline {
	return &Pipeline{
		eventBus:  eventbus.GetEventBus(),
		processor: processor,
	}
}

func (p *Pipeline) Start() {
	sub := p.eventBus.Subscribe(100)
	go p.processActions(sub)
}

func (p *Pipeline) processActions(ch <-chan eventbus.Event) {
	for event := range ch {
		if event.Type == eventbus.EventAction {
			action, ok := event.Payload.(Action)
			if !ok {
				continue
			}

			// Process with middleware chain
			eventbus.GetEventBus().Process(func(e eventbus.Event) error {
				return p.processor.ProcessAction(context.Background(), action)
			})(event)
		}
	}
}
