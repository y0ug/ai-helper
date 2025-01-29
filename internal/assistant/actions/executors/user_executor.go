package executors

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
	"github.com/y0ug/ai-helper/internal/assistant/ui"
)

type UserInteractionExecutor struct {
	pendingRequests sync.Map
	logger          *slog.Logger
	timeout         time.Duration
	eventBus        *eventbus.EventBus
	responseChan    chan actions.UserResponseAction
	ui              ui.UserInterface
}

type PendingRequest struct {
	OriginalAction actions.Action
	ReceivedAt     time.Time
	ResponseChan   chan actions.UserResponseAction
}

func NewUserInteractionExecutor(
	logger *slog.Logger,
	eventBus *eventbus.EventBus,
	ui ui.UserInterface,
) *UserInteractionExecutor {
	e := &UserInteractionExecutor{
		logger:       logger,
		timeout:      5 * time.Minute,
		eventBus:     eventBus,
		responseChan: make(chan actions.UserResponseAction),
	}

	sub := eventBus.Subscribe(100)
	go e.handleEvents(sub)

	// go e.processResponses()
	go e.cleanupRoutine()
	return e
}

func (e *UserInteractionExecutor) CanHandle(action actions.Action) bool {
	_, ok := action.Payload.(actions.UserConfirmAction)
	return ok
}

func (e *UserInteractionExecutor) Handle(
	ctx context.Context,
	action actions.Action,
) ([]actions.Action, error) {
	logger := actions.GetLogger(ctx)

	confirmAction := action.Payload.(actions.UserConfirmAction)

	requestID := uuid.New().String()
	responseChan := make(chan actions.UserResponseAction)

	now := time.Now()
	expiresAt := now.Add(e.timeout)

	// Store the pending request
	e.pendingRequests.Store(requestID, &PendingRequest{
		OriginalAction: confirmAction.ParentAction,
		ReceivedAt:     now,
		ResponseChan:   responseChan,
	})

	// Publish confirmation request event
	e.eventBus.Publish(eventbus.NewEvent(
		eventbus.EventUserConfirm,
		eventbus.UserConfirmRequest{
			ID:        requestID,
			Message:   "please confirm the action",
			Action:    confirmAction.ParentAction,
			ExpiresAt: expiresAt,
		},
	))

	// Wait for response with timeout
	select {
	case response := <-responseChan:
		// Update original action based on response
		switch originalAction := confirmAction.ParentAction.Payload.(type) {
		case actions.ShellCommandAction:
			if response.Allowed {
				newConfirmedaction := originalAction.WithConfirmed(&action)
				return []actions.Action{newConfirmedaction}, nil
			}
		default:
			logger.Error("Unsupported action type for confirmation",
				"action_type", confirmAction.ParentAction.Type)
			return nil, fmt.Errorf("unsupported action type for confirmation")
		}
	case <-time.After(e.timeout):
		e.pendingRequests.Delete(requestID)
		return nil, fmt.Errorf("confirmation timeout")
	case <-ctx.Done():
		e.pendingRequests.Delete(requestID)
		return nil, ctx.Err()
	}

	return nil, nil
}

func (e *UserInteractionExecutor) handleEvents(sub <-chan eventbus.Event) {
	for event := range sub {
		if event.Type == eventbus.EventUserResponse {
			response := event.Payload.(eventbus.UserResponse)
			if req, ok := e.pendingRequests.Load(response.ID); ok {
				pendingReq := req.(*PendingRequest)
				// Send response to the waiting Handle function
				pendingReq.ResponseChan <- actions.UserResponseAction{
					Allowed: response.Approved,
					// Input:   response.Input,
				}
				e.pendingRequests.Delete(response.ID)
			}
		}
	}
}

func (e *UserInteractionExecutor) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		e.pendingRequests.Range(func(key, value interface{}) bool {
			req := value.(*PendingRequest)
			if time.Since(req.ReceivedAt) > e.timeout {
				e.pendingRequests.Delete(key)
				e.logger.Warn("Cleaned up expired request",
					"age", time.Since(req.ReceivedAt),
					"action_id", req.OriginalAction.ID,
				)
				// Close response channel to prevent goroutine leak
				close(req.ResponseChan)
			}
			return true
		})
	}
}
