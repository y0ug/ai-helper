package executors

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/y0ug/ai-helper/internal/assistant/actions"
	"github.com/y0ug/ai-helper/internal/assistant/ui"
)

type UserInteractionExecutor struct {
	pendingRequests sync.Map
	uim             *ui.UIInteractionManager
	logger          *slog.Logger
	timeout         time.Duration
}

type PendingRequest struct {
	OriginalAction actions.Action
	ReceivedAt     time.Time
}

func NewUserInteractionExecutor(
	logger *slog.Logger,
	uim *ui.UIInteractionManager,
) *UserInteractionExecutor {
	e := &UserInteractionExecutor{
		logger:  logger,
		uim:     uim,
		timeout: 5 * time.Minute,
	}

	go e.processResponses()
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

	// Store using all context identifiers for redundancy
	key := fmt.Sprintf("%s|%s|%s",
		confirmAction.Context.ChainID,
		confirmAction.Context.ParentID,
		confirmAction.Context.ToolCallID,
	)

	logger.Debug("Storing pending request", "key", key)

	e.pendingRequests.Store(key, &PendingRequest{
		OriginalAction: confirmAction.ParentAction,
		ReceivedAt:     time.Now(),
	})

	logger.Debug("Stored pending request")

	// forward the confirmation request to the UIInteractionManager
	e.uim.ConfirmChan <- action
	return nil, nil
}

func (e *UserInteractionExecutor) processResponses() {
	logger := e.logger
	for action := range e.uim.ResponseChan {
		switch response := action.Payload.(type) {
		case actions.UserResponseAction:
			// Try multiple key formats for redundancy
			keys := []string{
				fmt.Sprintf("%s|%s|%s", response.Context.ChainID, response.Context.ParentID, response.Context.ToolCallID),
				fmt.Sprintf("|%s|%s", response.Context.ParentID, response.Context.ToolCallID),
				fmt.Sprintf("%s||", response.Context.ChainID),
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
				logger.Warn("Orphaned user response", "action", response)
			}
		default:
			logger.Warn("ResponChan action type not supported", "action", action)
		}
	}
}

func (e *UserInteractionExecutor) handleResponse(
	originalAction actions.Action,
	response actions.UserResponseAction,
) {
	logger := e.logger
	// Update original action based on response
	switch action := originalAction.Payload.(type) {
	case actions.ShellCommandAction:
		action.Confirmed = response.Allowed
		originalAction.Payload = action

		logger.Info("Re-enqueueing action with confirmation",
			"action_id", originalAction.ID,
			"confirmed", response.Allowed,
		)

		// Re-enqueue original action with updated state
		go func() {
			select {
			case e.uim.ActionChan <- originalAction:
			case <-time.After(1 * time.Second):
				e.logger.Error("Failed to re-enqueue confirmed action")
			}
		}()

	default:
		logger.Error("Unsupported action type for confirmation",
			"action_type", originalAction.Type)
	}
}

func (e *UserInteractionExecutor) cleanupRoutine() {
	// TODO: should we track the creation that come from New?
	// logger := actions.GetLogger(ctx)
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
