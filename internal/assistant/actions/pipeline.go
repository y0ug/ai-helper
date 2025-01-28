package actions

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
)

type Processor[T any] func(context.Context, T) ([]T, error)

type Pipeline struct {
	eventBus  *eventbus.EventBus
	cb        Processor[Action]
	runCtx    context.Context
	runCancel context.CancelFunc
	logger    *slog.Logger
}

func NewPipeline(logger *slog.Logger, cb Processor[Action]) *Pipeline {
	return &Pipeline{
		eventBus: eventbus.GetEventBus(),
		cb:       cb,
		logger:   logger,
	}
}

func (p *Pipeline) Start(ctx context.Context) {
	if p.runCtx == nil {
		p.runCtx, p.runCancel = context.WithCancel(ctx)
		sub := p.eventBus.Subscribe(100)
		go p.processActions(sub)
	}
}

func (p *Pipeline) processActions(ch <-chan eventbus.Event) {
	for event := range ch {
		if event.Type == eventbus.EventActionProcess {
			action, ok := event.Payload.(Action)
			if !ok {
				continue
			}

			p.logger.Debug("PIPELINE: Processing action", "action", action)

			p.eventBus.Process(func(e eventbus.Event) error {
				// We could track started/completed here?
				// p.eventBus.Publish(eventbus.NewEvent(
				// 	eventbus.EventActionStart,
				// 	actions.ActionTrace{
				// 		ActionID: action.ID,
				// 		Status:   "started",
				// 	},
				// ))

				results, err := p.cb(p.runCtx, action)
				if err != nil {
					p.logger.Error("PIPELINE: Error processing action", "error", err)
					return fmt.Errorf("error processing action: %w", err)
				}

				// Publish results/follows up to EventActionResult
				for _, result := range results {
					p.logger.Debug("PIPELINE: publishing result taction", "result", result)
					// result.Completed = true
					p.eventBus.Publish(eventbus.NewEvent(
						eventbus.EventAction,
						result,
					))
				}

				return nil
			})(event)

			p.logger.Debug("PIPELINE: Processing end", "action", action)
		}
	}
}
