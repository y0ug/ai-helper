package ui

import (
	"bufio"
	"fmt"
	"strings"
	"sync"

	"github.com/y0ug/ai-helper/internal/assistant/eventbus"
)

type UserInterface interface {
	Publish(eventbus.Event) error
	Output(eventbus.Event) error
	Error(eventbus.Event) error
	Status(eventbus.Event) error
	RequestConfirmation(eventbus.Event) (bool, error)
	Shutdown(eventbus.Event) error
	FileNotification(eventbus.Event) error
}

type CliUI struct {
	bus *eventbus.EventBus
	mu  sync.Mutex
}

var _ UserInterface = &CliUI{}

// NewCliUI creates a new CLI UI
func NewCliUI(bus *eventbus.EventBus) *CliUI {
	cli := &CliUI{
		bus: bus,
	}
	// go cli.listenEvents()
	return cli
}

func Publish(u UserInterface, event eventbus.Event) error {
	switch event.Type {
	case eventbus.EventOutput:
		return u.Output(event)
	case eventbus.EventError:
		return u.Error(event)
	case eventbus.EventStatusUpdate:
		return u.Status(event)
	case eventbus.EventUserConfirm:
		u.RequestConfirmation(event)

	case eventbus.EventFileNotification:
		return u.FileNotification(event)
	case eventbus.EventShutdown:
		return u.Shutdown(event)
	}
	return nil
}

func (u *CliUI) Publish(event eventbus.Event) error {
	return Publish(u, event)
}

func (u *CliUI) Output(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	message, ok := event.Payload.(string)
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	fmt.Print(message)
	return nil
}

func (u *CliUI) Error(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	errorMessage, ok := event.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	fmt.Printf("Error: %v\n", errorMessage["error"])
	return nil
}

func (u *CliUI) Status(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	errorMessage, ok := event.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	fmt.Printf("Error: %v\n", errorMessage["error"])
	return nil
}

func (u *CliUI) RequestConfirmation(event eventbus.Event) (bool, error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	payload, ok := event.Payload.(eventbus.UserConfirmRequest)
	if !ok {
		return false, fmt.Errorf("invalid payload type")
	}

	fmt.Printf("Confirmation [%s]: %s (y/n): ", payload.ID, payload.Message)
	return false, nil
}

func (u *CliUI) FileNotification(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	payload, ok := event.Payload.(eventbus.FileOperation)
	if !ok {
		return fmt.Errorf("invalid payload type")
	}

	fmt.Printf("File FileOperation %s: %s\n", payload.Type, payload.Files)
	return nil
}

func (u *CliUI) Shutdown(event eventbus.Event) (err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	fmt.Printf("Shutdown\n")
	return nil
}

// func (c *CliUI) listenEvents() {
// 	sub := c.bus.Subscribe(100)
// 	defer c.bus.Unsubscribe(sub)
//
// 	scanner := bufio.NewScanner(os.Stdin)
//
// 	for evt := range sub {
// 		switch evt.Type {
// 		case eventbus.EventOutput:
// 			c.ShowOutput(.Message)
// 		case EventShowError:
// 			var e ShowErrorEvent
// 			json.Unmarshal(evt.Payload, &e)
// 			c.ShowError(errors.New(e.Error))
// 		case EventShowStatus:
// 			var e ShowStatusEvent
// 			json.Unmarshal(evt.Payload, &e)
// 			c.ShowStatus(e.Status)
// 		case EventRequestConfirmation:
// 			var e RequestConfirmationEvent
// 			json.Unmarshal(evt.Payload, &e)
// 			approved, err := c.handleConfirmation(e.RequestID, e.Message, scanner)
// 			if err != nil {
// 				// Optionally, handle the error (e.g., log it)
// 				continue
// 			}
// 			// Publish the confirmation response
// 			resp := ConfirmationResponseEvent{
// 				RequestID: e.RequestID,
// 				Approved:  approved,
// 			}
// 			c.bus.Publish(EventConfirmationResponse, resp)
// 		}
// 	}
// }

func (c *CliUI) handleConfirmation(
	requestID, message string,
	scanner *bufio.Scanner,
) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for {
		fmt.Printf("Confirmation [%s]: %s (y/n): ", requestID, message)
		if !scanner.Scan() {
			return false, fmt.Errorf("failed to read input")
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "y" || input == "Y" {
			return true, nil
		} else if input == "n" || input == "N" {
			return false, nil
		} else {
			fmt.Println("Please enter 'y' or 'n'.")
		}
	}
}
