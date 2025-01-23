package editservice

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
)

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	return reflector.Reflect(v)
}

func StrToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type Tooler interface {
	Execute(ctx context.Context, input json.RawMessage) (interface{}, error)
	GetName() string
	GetChatTool() chat.Tool
}

type ToolHandler[T any] func(ctx context.Context, input T) (interface{}, error)

type ToolBase[T any] struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema interface{}    `json:"input_schema,omitempty"`
	handler     ToolHandler[T] `json:"-"`
}

func (t ToolBase[T]) GetName() string {
	return t.Name
}

func (t ToolBase[T]) GetChatTool() chat.Tool {
	return chat.Tool{
		Name:        t.Name,
		Description: StrToPtr(t.Description),
		InputSchema: t.InputSchema,
	}
}

func (t ToolBase[T]) Execute(ctx context.Context, input json.RawMessage) (interface{}, error) {
	var args T
	err := json.Unmarshal(input, &args)
	if err != nil {
		return nil, fmt.Errorf("%s failed to unmarshal input: %w", t.Name, err)
	}

	result, err := t.handler(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("%s failed to execute: %w", t.Name, err)
	}
	return result, nil
}

func NewTool[T any](
	name string,
	description string,
	handler ToolHandler[T],
) *ToolBase[T] {
	inputSchema := GenerateSchema[T]()
	return &ToolBase[T]{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
		handler:     handler,
	}
}
