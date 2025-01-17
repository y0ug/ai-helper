package llmagent

import (
	"context"

	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/mcpclient"
)

type ToolHandler func(ctx context.Context, input map[string]interface{}) ([]interface{}, error)

func GetToolHandler(c mcpclient.MCPClientInterface, name string) ToolHandler {
	return func(ctx context.Context, input map[string]interface{}) ([]interface{}, error) {
		result, err := c.CallTool(ctx, name, input)
		if err != nil {
			return nil, err
		}
		return result.Content, nil
	}
}

func MCPClientToolToTool(tools ...mcpclient.Tool) []chat.Tool {
	result := make([]chat.Tool, 0)
	for _, tool := range tools {
		result = append(result, chat.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}
	return result
}
