package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/pkg/highlighter"
)

type UIInteractionManager struct {
	// Communication channels
	InputChan          chan string         // User input from console
	OutputChan         chan string         // Streamed LLM output
	ConfirmChan        chan actions.Action // Actions requiring user input
	ResponseChan       chan actions.Action // this channel is handle by UserInteractionExector to recevied and requeue the action if confirmed
	ActionChan         chan actions.Action // Channel to push to the assistant action queue, push UserInteractionExecutor from and get in
	StatusChan         chan string         // Status updates
	ErrorChan          chan error          // Error notifications
	ShutdownChan       chan struct{}       // Shutdown signal
	pendingConfirm     chan bool
	assistantInputChan chan string

	// State management
	mu           sync.Mutex
	active       bool
	highlight    *highlighter.Highlighter
	outputWriter io.Writer
}

func NewUIInteractionManager(h *highlighter.Highlighter) *UIInteractionManager {
	return &UIInteractionManager{
		InputChan:          make(chan string, 10),
		OutputChan:         make(chan string, 100),
		ConfirmChan:        make(chan actions.Action, 10),
		ResponseChan:       make(chan actions.Action, 10),
		ActionChan:         make(chan actions.Action, 10),
		StatusChan:         make(chan string, 20),
		ErrorChan:          make(chan error, 10),
		ShutdownChan:       make(chan struct{}),
		assistantInputChan: make(chan string, 10),
		pendingConfirm:     make(chan bool),
		highlight:          h,
		active:             true,
	}
}

func (uim *UIInteractionManager) GetAssistantInputChan() chan string {
	return uim.assistantInputChan
}

func (uim *UIInteractionManager) RequestConfirmation(prompt string) <-chan bool {
	uim.OutputChan <- prompt + " [y/N] "
	return uim.pendingConfirm
}

func (uim *UIInteractionManager) Start() {
	go uim.handleOutput()
	go uim.handleStatus()
	go uim.handleErrors()
	go uim.handleConfirmations()
	go uim.routeInputToAssistant()
}

func (uim *UIInteractionManager) HandleInput(input string) {
	isConfirmed := strings.ToLower(input) == "y"
	fmt.Printf("isConfirmed: %v\n", isConfirmed)
	select {
	case uim.pendingConfirm <- isConfirmed:
		fmt.Printf("uim.pendingConfirm: %v\n", uim.pendingConfirm)
	default:
		// No pending confirmation, process normally
		uim.InputChan <- input
	}
}

func (uim *UIInteractionManager) routeInputToAssistant() {
	for {
		select {
		case input := <-uim.InputChan:
			// Validate input before forwarding
			if cleaned := strings.TrimSpace(input); cleaned != "" {
				uim.OutputChan <- "⌛ Processing input..."
				uim.assistantInputChan <- cleaned
			}
		case <-uim.ShutdownChan:
			return
		}
	}
}

func (uim *UIInteractionManager) StreamOutput(h *highlighter.Highlighter) {
	uim.mu.Lock()
	defer uim.mu.Unlock()
	uim.highlight = h
	uim.outputWriter = h
}

func (uim *UIInteractionManager) Shutdown() {
	uim.mu.Lock()
	defer uim.mu.Unlock()

	if uim.active {
		close(uim.ShutdownChan)
		uim.active = false
	}
}

func (uim *UIInteractionManager) handleOutput() {
	for {
		select {
		case output := <-uim.OutputChan:
			uim.writeOutput(output)
		case <-uim.ShutdownChan:
			return
		}
	}
}

func (uim *UIInteractionManager) handleStatus() {
	for {
		select {
		case status := <-uim.StatusChan:
			uim.writeOutput(fmt.Sprintf("\n[STATUS] %s\n", status))
		case <-uim.ShutdownChan:
			return
		}
	}
}

func (uim *UIInteractionManager) handleErrors() {
	for {
		select {
		case err := <-uim.ErrorChan:
			uim.writeOutput(fmt.Sprintf("\n[ERROR] %v\n", err))
		case <-uim.ShutdownChan:
			return
		}
	}
}

func (uim *UIInteractionManager) handleConfirmations() {
	for {
		select {
		case action := <-uim.ConfirmChan:
			uim.processConfirmation(action)
		case <-uim.ShutdownChan:
			return
		}
	}
}

func (uim *UIInteractionManager) processConfirmation(action actions.Action) {
	switch a := action.Payload.(type) {
	case actions.UserConfirmAction:
		fmt.Printf("received: %v\n", a)
		fmt.Printf("received: %v\n", action)
		response := actions.NewUserResponseAction(a.ParentAction, <-uim.RequestConfirmation(a.Question)).
			WithParent(&action)
		fmt.Printf("response: %v\n", response)
		uim.ResponseChan <- response
	}
}

func (uim *UIInteractionManager) writeOutput(content string) {
	uim.mu.Lock()
	defer uim.mu.Unlock()

	if uim.outputWriter != nil {
		fmt.Fprint(uim.outputWriter, content)
	}
}

// Public API Methods
func (uim *UIInteractionManager) DisplayMessage(msg string) {
	uim.OutputChan <- msg
}

func (uim *UIInteractionManager) DisplayError(err error) {
	uim.ErrorChan <- err
}

func (uim *UIInteractionManager) UpdateStatus(status string) {
	uim.StatusChan <- status
}

func (uim *UIInteractionManager) GetInput() <-chan string {
	return uim.InputChan
}

func (uim *UIInteractionManager) IsActive() bool {
	uim.mu.Lock()
	defer uim.mu.Unlock()
	return uim.active
}
