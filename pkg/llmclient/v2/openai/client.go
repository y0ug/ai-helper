package openai

import (
	"os"

	base "github.com/y0ug/ai-helper/pkg/llmclient/v2/base"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
)

// Client creates a struct with services and top level methods that help with
// interacting with the openai API. You should not instantiate this client
// directly, and instead use the [NewClient] method instead.
type Client struct {
	*base.BaseClient
	Chat *ChatCompletionService
}

// NewClient generates a new client with the default option read from the
// environment (OPENAI_API_KEY, OPENAI_ORG_ID, OPENAI_PROJECT_ID). The option
// passed in as arguments are applied after these default arguments, and all option
// will be passed down to the services and requests that this client makes.
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

	r.Chat = NewChatCompletionService(r.BaseClient.Options...)

	return
}
