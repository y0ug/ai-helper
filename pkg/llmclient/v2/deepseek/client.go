package gemini

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/y0ug/ai-helper/pkg/llmclient/v2/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestconfig"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
)

type Client struct {
	openai.Client
}

type DeepSeekUsage struct {
	PromptTokens          int `json:"prompt_tokens"`
	CompletionTokens      int `json:"completion_tokens"`
	TotalTokens           int `json:"total_tokens"`
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
}
type ChatCompletion struct {
	// Embed all the OpenAI fields.
	openai.ChatCompletion

	// Override usage with your DeepSeek usage struct
	Usage DeepSeekUsage `json:"usage"`
}

// Custom Unmarshal to fill our DeepSeekUsage, while still reusing the embedded fields.
func (r *ChatCompletion) UnmarshalJSON(data []byte) error {
	type Alias ChatCompletion
	// This unmarshal step will fill all fields, including our new `Usage`
	if err := json.Unmarshal(data, (*Alias)(r)); err != nil {
		return err
	}
	return nil
}

func WithEnvironmentProduction() requestoption.RequestOption {
	return requestoption.WithBaseURL("https://api.deepseek.com/v1/")
}

func NewClient(opts ...requestoption.RequestOption) (r *Client) {
	r = &Client{}
	defaults := []requestoption.RequestOption{
		WithEnvironmentProduction(),
	}
	if o, ok := os.LookupEnv("DEEPSEEK_API_KEY"); ok {
		defaults = append(defaults, requestoption.WithAuthToken(o))
	}
	opts = append(defaults, opts...)
	r = &Client{
		openai.Client{
			Options:  append(defaults, opts...),
			NewError: openai.NewAPIErrorOpenAI,
		},
	}

	r.Chat = NewChatCompletionService(r.Options...)
	return r
}

// Wrap the OpenAI ChatCompletionService in our own
type ChatCompletionService struct {
	*openai.ChatCompletionService
}

// Create our custom service that reuses openai.NewChatCompletionService under the hood
func NewChatCompletionService(opts ...requestoption.RequestOption) *ChatCompletionService {
	return &ChatCompletionService{
		ChatCompletionService: openai.NewChatCompletionService(opts...),
	}
}

// We only override the response type in this call
func (svc *ChatCompletionService) New(
	ctx context.Context,
	body openai.ChatCompletionNewParams,
	opts ...requestoption.RequestOption,
) (*ChatCompletion, error) {
	opts = append(svc.Options, opts...)
	path := "chat/completions"

	// We want to unmarshal into *ChatCompletion (which has DeepSeekUsage)
	var res *ChatCompletion
	err := requestconfig.ExecuteNewRequest(
		ctx,
		http.MethodPost,
		path,
		body,
		&res,
		openai.NewAPIErrorOpenAI,
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return res, nil
}
