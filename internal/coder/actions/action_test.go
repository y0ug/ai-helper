package actions

import (
	"fmt"
	"strings"
	"testing"

	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

func ToGenericAction[T ActionPayload](a Action[T]) Action[any] {
	return Action[any]{
		Type:    a.Type,
		Payload: a.Payload,
	}
}

func ActionHandler(level int, actions ...Action[any]) {
	tab := strings.Repeat("\t", level)
	for _, action := range actions {
		payload := action.Payload
		switch v := payload.(type) {
		case ApplyEdit:
			fmt.Printf("%sApplyEdit\t%s\n", tab, v.Filename)
		case ToolResultAction: // Changed from ToolResultAction[interface{}]
			fmt.Printf("%sToolResultAction\t%s\n", tab, v.ToolResult.ToolUseID)
			ActionHandler(level+1, v.NextAction)
		// case ToolResultAction[ApplyEdit]:
		// 	fmt.Printf("%sToolResultAction[ApplyEdit]\t%s\n", tab, v.ToolResult.ID)
		// 	ActionHandler(level+1, ToGenericAction(v.NextAction))

		default:
			fmt.Printf("%sUnknown action type: %T\n", tab, payload)
		}
	}
}

func TestAction(t *testing.T) {
	t.Run("TestNewAction", func(t *testing.T) {
		editAction := NewParsedAction(ApplyEdit{
			Filename: "test.txt",
			Original: "Hello world",
			Updated:  "Hello Go",
		})

		toolResultAction := NewParsedAction(ToolResultAction{
			ToolResult: *chat.NewToolResultContent("1234", ""),
			NextAction: editAction,
		})

		// fmt.Println(editAction)
		// fmt.Println(toolResultAction)

		queue := make([]Action[any], 0)

		queue = append(queue, editAction)
		queue = append(queue, toolResultAction)

		for _, action := range queue {
			ActionHandler(0, action)
		}
	})
}
