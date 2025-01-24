package executors

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
)

type UserInteractionExecutor struct {
	pendingRequests sync.Map // map[string]*PendingRequest
	responseChan    <-chan UserResponse
	actionChan      chan<- actions.Action // Add this
	logger          *slog.Logger
	timeout         time.Duration
}

type PendingRequest struct {
	OriginalAction actions.Action
	ReceivedAt     time.Time
}

type UserResponse struct {
	ChainID    string
	ParentID   string
	ToolCallID string
	Allowed    bool
}

func NewUserInteractionExecutor(
	responseChan <-chan UserResponse,
	actionChan chan<- actions.Action, // Add this parameter
	logger *slog.Logger,
) *UserInteractionExecutor {
	e := &UserInteractionExecutor{
		responseChan: responseChan,
		actionChan:   actionChan,
		logger:       logger,
		timeout:      5 * time.Minute,
	}

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
	confirmAction := action.Payload.(actions.UserConfirmAction)

	// Store using all context identifiers for redundancy
	key := fmt.Sprintf("%s|%s|%s",
		action.Context.ChainID,
		action.Context.ParentID,
		action.Context.ToolCallID,
	)

	e.pendingRequests.Store(key, &PendingRequest{
		OriginalAction: actions.Action{
			ID:      action.Context.ParentID, // Original action's ID
			Context: confirmAction.Context,
		},
		ReceivedAt: time.Now(),
	})

	e.logger.Debug("Stored pending request",
		"chain_id", action.Context.ChainID,
		"parent_id", action.Context.ParentID,
		"tool_call_id", action.Context.ToolCallID,
	)

	return nil, nil
}

func (e *UserInteractionExecutor) processResponses() {
	for response := range e.responseChan {
		// Try multiple key formats for redundancy
		keys := []string{
			fmt.Sprintf("%s|%s|%s", response.ChainID, response.ParentID, response.ToolCallID),
			fmt.Sprintf("|%s|%s", response.ParentID, response.ToolCallID),
			fmt.Sprintf("%s||", response.ChainID),
		}

		var foundAction *actions.Action
		for _, key := range keys {
			if req, ok := e.pendingRequests.Load(key); ok {
				foundAction = &req.(*PendingRequest).OriginalAction
				e.pendingRequests.Delete(key)
				break
			}
		}

		if foundAction != nil {
			e.handleResponse(*foundAction, response)
		} else {
			e.logger.Warn("Orphaned user response",
				"chain_id", response.ChainID,
				"parent_id", response.ParentID,
			)
		}
	}
}

func (e *UserInteractionExecutor) handleResponse(
	originalAction actions.Action,
	response UserResponse,
) {
	// Update original action based on response
	switch action := originalAction.Payload.(type) {
	case actions.ShellCommandAction:
		action.Confirmed = response.Allowed
		originalAction.Payload = action

		e.logger.Info("Re-enqueueing action with confirmation",
			"action_id", originalAction.ID,
			"confirmed", response.Allowed,
		)

		// Re-enqueue original action with updated state
		go func() {
			select {
			case e.actionChan <- originalAction:
			case <-time.After(1 * time.Second):
				e.logger.Error("Failed to re-enqueue confirmed action")
			}
		}()

	default:
		e.logger.Error("Unsupported action type for confirmation",
			"action_type", originalAction.Type)
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
			}
			return true
		})
	}
}
