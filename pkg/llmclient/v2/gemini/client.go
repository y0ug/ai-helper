package gemini

import (
	"os"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
)

type Client struct {
	openai.Client
}

func WithEnvironmentProduction() requestoption.RequestOption {
	return requestoption.WithBaseURL("https://generativelanguage.googleapis.com/v1beta/openai/")
}

func NewClient(opts ...requestoption.RequestOption) (r *Client) {
	r = &Client{}
	defaults := []requestoption.RequestOption{
		WithEnvironmentProduction(),
	}
	if o, ok := os.LookupEnv("GEMINI_API_KEY"); ok {
		defaults = append(defaults, requestoption.WithAuthToken(o))
	}
	opts = append(defaults, opts...)
	r = &Client{
		openai.Client{
			Options:  append(defaults, opts...),
			NewError: openai.NewAPIErrorOpenAI,
		},
	}

	r.Chat = openai.NewChatCompletionService(r.Options...)

	return r
}
