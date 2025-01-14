package openrouter

import (
	"os"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
)

type Client struct {
	*openai.Client
}

func WithEnvironmentProduction() requestoption.RequestOption {
	return requestoption.WithBaseURL("https://openrouter.ai/api/v1/")
}

func NewClient(opts ...requestoption.RequestOption) (r *Client) {
	defaults := []requestoption.RequestOption{
		WithEnvironmentProduction(),
	}
	if o, ok := os.LookupEnv("OPENROUTER_API_KEY"); ok {
		defaults = append(defaults, requestoption.WithAuthToken(o))
	}
	opts = append(defaults, opts...)

	r = &Client{
		Client: openai.NewClient(opts...),
	}

	return r
}