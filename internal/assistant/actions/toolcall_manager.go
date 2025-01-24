package actions

import (
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ToolCallState int

const (
	StatePending ToolCallState = iota
	StateProcessing
	StateCompleted
	StateFailed
)

type ToolCallRecord struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	State     ToolCallState
	Actions   []Action
	Results   []string
}

type ToolCallManager struct {
	activeCalls sync.Map // map[string]*ToolCallRecord
	logger      *slog.Logger
}

func NewToolCallManager(logger *slog.Logger) *ToolCallManager {
	return &ToolCallManager{
		logger: logger,
	}
}

func (m *ToolCallManager) StartToolCall(toolCallID string) {
	record := &ToolCallRecord{
		ID:        toolCallID,
		CreatedAt: time.Now(),
		State:     StatePending,
	}
	m.activeCalls.Store(toolCallID, record)
}

func (m *ToolCallManager) AddAction(toolCallID string, action Action) {
	if record, ok := m.activeCalls.Load(toolCallID); ok {
		rec := record.(*ToolCallRecord)
		rec.Actions = append(rec.Actions, action)
		rec.State = StateProcessing
		m.activeCalls.Store(toolCallID, rec)
	}
}

func (m *ToolCallManager) CompleteToolCall(toolCallID string, result string) {
	if record, ok := m.activeCalls.Load(toolCallID); ok {
		rec := record.(*ToolCallRecord)
		rec.Results = append(rec.Results, result)
		rec.State = StateCompleted
		rec.UpdatedAt = time.Now()
		m.activeCalls.Store(toolCallID, rec)
	}
}

func (m *ToolCallManager) GetToolCallState(toolCallID string) (*ToolCallRecord, bool) {
	record, ok := m.activeCalls.Load(toolCallID)
	if !ok {
		return nil, false
	}
	return record.(*ToolCallRecord), true
}

func (m *ToolCallManager) GetActionChain(chainID uuid.UUID) []Action {
	var chain []Action
	m.activeCalls.Range(func(_, v interface{}) bool {
		action := v.(Action)
		if action.Context.ChainID == chainID {
			chain = append(chain, action)
		}
		return true
	})
	return chain
}
