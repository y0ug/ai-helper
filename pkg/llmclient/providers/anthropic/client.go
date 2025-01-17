package anthropic

import (
	"os"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/client"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
)

type Client struct {
	*client.BaseClient
	Message *MessageService
}

func NewClient(opts ...options.RequestOption) (r *Client) {
	defaults := []options.RequestOption{
		WithEnvironmentProduction(), WithApiVersionAnthropic(),
	}
	if o, ok := os.LookupEnv("ANTHROPIC_API_KEY"); ok {
		defaults = append(defaults, options.WithApiKey("x-api-key", o))
	}
	r = &Client{
		BaseClient: &client.BaseClient{
			Options:  append(defaults, opts...),
			NewError: NewAPIErrorAnthropic,
		},
	}

	r.Message = NewMessageService(r.BaseClient.Options...)

	return
}
