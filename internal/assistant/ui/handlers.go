package ui

import (
	"context"
	"fmt"

	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
	"github.com/y0ug/ai-helper/pkg/highlighter"
)

type InputHandler struct {
	eventBus         *eventbus.EventBus
	ctx              context.Context
	cancel           context.CancelFunc
	consoleInputChan chan string
}

func NewInputHandler() *InputHandler {
	ctx, cancel := context.WithCancel(context.Background())
	return &InputHandler{
		consoleInputChan: make(chan string),
		eventBus:         eventbus.GetEventBus(),
		ctx:              ctx,
		cancel:           cancel,
	}
}

func (h *InputHandler) Start() {
	go h.listenConsole()
	go h.listenAPI()
}

func (h *InputHandler) listenConsole() {
	// Integration with console input
	for {
		select {
		case <-h.ctx.Done():
			return
		case input := <-h.consoleInputChan:
			h.eventBus.Publish(eventbus.NewEvent(
				eventbus.EventInput,
				eventbus.UserInput{Source: "console", Content: input},
			))
		}
	}
}

func (h *InputHandler) listenAPI() {
	// WebSocket/HTTP API integration
}

type OutputHandler struct {
	eventBus    *eventbus.EventBus
	highlighter *highlighter.Highlighter
}

func NewOutputHandler(h *highlighter.Highlighter) *OutputHandler {
	return &OutputHandler{
		eventBus:    eventbus.GetEventBus(),
		highlighter: h,
	}
}

func (h *OutputHandler) Start() {
	sub := h.eventBus.Subscribe(100)
	go h.processOutput(sub)
}

func (h *OutputHandler) processOutput(ch <-chan eventbus.Event) {
	for event := range ch {
		if event.Type == eventbus.EventOutput {
			output, ok := event.Payload.(string)
			if !ok {
				continue
			}

			formatted := output

			fmt.Fprint(h.highlighter, formatted)
			// // Handle different output types
			// switch event.Metadata["output_type"] {
			// case "stream":
			// case "final":
			// 	fmt.Println("\n" + formatted)
			// case "error":
			// 	fmt.Fprintf(os.Stderr, "ERROR: %s", formatted)
			// }
		}
	}
}
