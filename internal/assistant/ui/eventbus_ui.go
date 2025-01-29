package ui

import (
	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
)

type EventBusUI struct {
	bus *eventbus.EventBus
}

var _ UserInterface = &EventBusUI{}

func NewEventBusUI(bus *eventbus.EventBus) *EventBusUI {
	return &EventBusUI{
		bus: bus,
	}
}

func (u *EventBusUI) Publish(event eventbus.Event) error {
	return Publish(u, event)
}

func (u *EventBusUI) Output(event eventbus.Event) (err error) {
	u.bus.Publish(event)
	return nil
}

func (u *EventBusUI) Error(event eventbus.Event) (err error) {
	u.bus.Publish(event)
	return nil
}

func (u *EventBusUI) Status(event eventbus.Event) (err error) {
	u.bus.Publish(event)
	return nil
}

func (u *EventBusUI) RequestConfirmation(event eventbus.Event) (bool, error) {
	u.bus.Publish(event)
	return false, nil
}

func (u *EventBusUI) FileNotification(event eventbus.Event) (err error) {
	u.bus.Publish(event)
	return nil
}

func (u *EventBusUI) Shutdown(event eventbus.Event) (err error) {
	u.bus.Publish(event)
	return nil
}
