package openai

import (
	"os"

	base "github.com/y0ug/ai-helper/pkg/llmclient/v2/base"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
)

type Client struct {
	*base.BaseClient
	Chat *ChatCompletionService
}

func NewClient(opts ...requestoption.RequestOption) (r *Client) {
	defaults := []requestoption.RequestOption{
		WithEnvironmentProduction(),
	}
	if o, ok := os.LookupEnv("OPENAI_API_KEY"); ok {
		defaults = append(defaults, requestoption.WithAuthToken(o))
	}
	if o, ok := os.LookupEnv("OPENAI_ORG_ID"); ok {
		defaults = append(defaults, WithOrganization(o))
	}
	if o, ok := os.LookupEnv("OPENAI_PROJECT_ID"); ok {
		defaults = append(defaults, WithProject(o))
	}
	r = &Client{
		BaseClient: &base.BaseClient{
			Options:  append(defaults, opts...),
			NewError: NewAPIErrorOpenAI,
		},
	}

	r.Chat = NewChatCompletionService(r.Options...)

	return
}
