package actions

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ActionChain struct {
	ChainID   uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
	Actions   []Action
	Results   []string
}

type ActionManager struct {
	activeChains sync.Map // map[uuid.UUID]*ActionChain
	logger       *slog.Logger
}

func NewActionManager(logger *slog.Logger) *ActionManager {
	return &ActionManager{
		logger: logger,
	}
}

func (m *ActionManager) RegisterAction(action Action) {
	chainID := action.Context.ChainID

	record, loaded := m.activeChains.LoadOrStore(chainID, &ActionChain{
		ChainID:   chainID,
		CreatedAt: time.Now(),
		Actions:   []Action{action},
	})

	if loaded {
		chain := record.(*ActionChain)
		chain.Actions = append(chain.Actions, action)
		chain.UpdatedAt = time.Now()
		m.activeChains.Store(chainID, chain)
	}
}

func (m *ActionManager) GetChain(chainID uuid.UUID) (*ActionChain, bool) {
	record, ok := m.activeChains.Load(chainID)
	if !ok {
		return nil, false
	}
	return record.(*ActionChain), true
}

func (m *ActionManager) GetAllChains() []*ActionChain {
	var chains []*ActionChain
	m.activeChains.Range(func(_, v interface{}) bool {
		chains = append(chains, v.(*ActionChain))
		return true
	})
	return chains
}

func (m *ActionManager) AddResult(chainID uuid.UUID, result string) {
	if record, ok := m.activeChains.Load(chainID); ok {
		chain := record.(*ActionChain)
		chain.Results = append(chain.Results, result)
		chain.UpdatedAt = time.Now()
		m.activeChains.Store(chainID, chain)
	}
}

func FindRootAction(m *ActionManager, action Action) Action {
	current := action
	for {
		if current.Context.ParentID == uuid.Nil {
			return current
		}

		if chain, ok := m.GetChain(current.Context.ChainID); ok {
			for _, a := range chain.Actions {
				if a.ID == current.Context.ParentID {
					current = a
					break
				}
			}
		}
	}
}

// // In actions/action_manager.go
// func (m *ActionManager) AddError(chainID uuid.UUID, err error) {
// 	if record, ok := m.activeChains.Load(chainID); ok {
// 		chain := record.(*ActionChain)
// 		chain.Results = append(chain.Results, fmt.Sprintf("ERROR: %v", err))
// 		chain.UpdatedAt = time.Now()
// 		m.activeChains.Store(chainID, chain)
// 	}
// }

func (m *ActionManager) AddError(chainID uuid.UUID, action Action, err error) {
	errMsg := fmt.Sprintf("ERROR: action_id=%s - %v", action.ID, err)
	m.AddResult(chainID, errMsg)
}
