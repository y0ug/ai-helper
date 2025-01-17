package anthropic

import (
	"os"

	"github.com/y0ug/ai-helper/pkg/llmclient/http/client"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/requestoption"
)

type Client struct {
	*client.BaseClient
	Message *MessageService
}

func NewClient(opts ...requestoption.RequestOption) (r *Client) {
	defaults := []requestoption.RequestOption{
		WithEnvironmentProduction(), WithApiVersionAnthropic(),
	}
	if o, ok := os.LookupEnv("ANTHROPIC_API_KEY"); ok {
		defaults = append(defaults, requestoption.WithApiKey("x-api-key", o))
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
