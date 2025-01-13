package openai

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/y0ug/ai-helper/pkg/llmclient/openai/requestconfig"
	"github.com/y0ug/ai-helper/pkg/llmclient/openai/requestoption"
)

type ChatCompletionChoice struct {
	FinishReason string                `json:"finish_reason,required"`
	Index        int64                 `json:"index,required"`
	Message      ChatCompletionMessage `json:"message,required"`
	JSON         string                `json:"-"`
}

func (r *ChatCompletionChoice) UnmarshalJSON(data []byte) (err error) {
	r.JSON = string(data)
	type Alias ChatCompletionChoice
	return json.Unmarshal(data, (*Alias)(r))
}

type ChatCompletionMessage struct {
	Role       string        `json:"role"`
	Refusal    string        `json:"refusal,omitempty"`
	Name       string        `json:"name,omitempty"`
	Audio      interface{}   `json:"audio,omitempty"`
	ToolCalls  []interface{} `json:"tool_calls,omitempty"`
	Content    string        `json:"content,omitempty"`
	ToolCallId string        `json:"tool_call_id,omitempty"`
	JSON       string        `json:"-"`
}

func (r *ChatCompletionMessage) UnmarshalJSON(data []byte) (err error) {
	r.JSON = string(data)
	type Alias ChatCompletionMessage
	return json.Unmarshal(data, (*Alias)(r))
}

type ChatCompletion struct {
	// A unique identifier for the chat completion.
	ID string `json:"id,required"`
	// A list of chat completion choices. Can be more than one if `n` is greater
	// than 1.
	Choices []ChatCompletionChoice `json:"choices,required"`
	// The Unix timestamp (in seconds) of when the chat completion was created.
	Created int64 `json:"created,required"`
	// The model used for the chat completion.
	Model string `json:"model,required"`
	// The object type, which is always `chat.completion`.
	Object string `json:"object,required"`
	// The service tier used for processing the request. This field is only included if
	// the `service_tier` parameter is specified in the request.
	ServiceTier interface{} `json:"service_tier,nullable"`
	// This fingerprint represents the backend configuration that the model runs with.
	//
	// Can be used in conjunction with the `seed` request parameter to understand when
	// backend changes have been made that might impact determinism.
	SystemFingerprint string `json:"system_fingerprint"`
	// Usage statistics for the completion request.
	Usage CompletionUsage `json:"usage"`
	JSON  string          `json:"-"`
}

func (r *ChatCompletion) UnmarshalJSON(data []byte) (err error) {
	r.JSON = string(data)
	type Alias ChatCompletion
	return json.Unmarshal(data, (*Alias)(r))
}

type CompletionUsage struct {
	CompletionTokens        int `json:"completion_tokens"`
	PromptTokens            int `json:"prompt_tokens"`
	TotalTokens             int `json:"total_tokens"`
	CompletionTokensDetails struct {
		AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
		AudioTokens              int `json:"audio_tokens"`
		ReasoningTokens          int `json:"reasoning_tokens"`
		RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
	}
	PromptTokensDetails struct {
		CachedTokens int `json:"cached_tokens"`
		AutdioTokens int `json:"audio_tokens"`
	} `json:"prompt_tokens_details"`
	Cost float64 `json:"cost,omitempty"`
}

type ChatCompletionService struct {
	Options []requestoption.RequestOption
}

// NewChatCompletionService generates a new service that applies the given options
// to each request. These options are applied after the parent client's options (if
// there is one), and before any request-specific options.
func NewChatCompletionService(opts ...requestoption.RequestOption) (r *ChatCompletionService) {
	r = &ChatCompletionService{}
	r.Options = opts
	return
}

// Creates a model response for the given chat conversation. Learn more in the
// [text generation](https://platform.openai.com/docs/guides/text-generation),
// [vision](https://platform.openai.com/docs/guides/vision), and
// [audio](https://platform.openai.com/docs/guides/audio) guides.
//
// Parameter support can differ depending on the model used to generate the
// response, particularly for newer reasoning models. Parameters that are only
// supported for reasoning models are noted below. For the current state of
// unsupported parameters in reasoning models,
// [refer to the reasoning guide](https://platform.openai.com/docs/guides/reasoning).
func (r *ChatCompletionService) New(
	ctx context.Context,
	body ChatCompletionNewParams,
	opts ...requestoption.RequestOption,
) (res *ChatCompletion, err error) {
	opts = append(r.Options[:], opts...)
	path := "chat/completions"
	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &res, opts...)
	return
}

// Creates a model response for the given chat conversation. Learn more in the
// [text generation](https://platform.openai.com/docs/guides/text-generation),
// [vision](https://platform.openai.com/docs/guides/vision), and
// [audio](https://platform.openai.com/docs/guides/audio) guides.
//
// Parameter support can differ depending on the model used to generate the
// response, particularly for newer reasoning models. Parameters that are only
// supported for reasoning models are noted below. For the current state of
// unsupported parameters in reasoning models,
// [refer to the reasoning guide](https://platform.openai.com/docs/guides/reasoning).
// func (r *ChatCompletionService) NewStreaming(
// 	ctx context.Context,
// 	body ChatCompletionNewParams,
// 	opts ...requestoption.RequestOption,
// ) (stream *ssestream.Stream[ChatCompletionChunk]) {
// 	var (
// 		raw *http.Response
// 		err error
// 	)
// 	opts = append(r.Options[:], opts...)
// 	opts = append([]requestoption.RequestOption{requestoption.WithJSONSet("stream", true)}, opts...)
// 	path := "chat/completions"
// 	err = requestconfig.ExecuteNewRequest(ctx, http.MethodPost, path, body, &raw, opts...)
// 	return ssestream.NewStream[ChatCompletionChunk](ssestream.NewDecoder(raw), err)
// }

type ChatCompletionNewParams struct {
	Model               string `json:"model"`
	MaxCompletionTokens *int   `json:"max_completion_tokens,omitempty"`
	ReasoningEffort     string `json:"reasoning_effort,omitempty"` // low, medium, high
	// Number between -2.0 and 2.0. Positive values penalize new tokens based on their existing frequency in the text so far, decreasing the model's likelihood to repeat the same line verbatim.
	FrequencyPenalty *float64     `json:"frequency_penalty,omitempty"`
	N                *int         `json:"n,omitempty"` // Number of completions to generate for each prompt.
	ResponseFormat   *interface{} `json:"response_format,omitempty"`
	Stop             *string      `json:"stop,omitempty"`   // Up to 4 sequences where the API will stop generating further tokens.
	Stream           bool         `json:"stream,omitempty"` // If true, the API will return a response as soon as it becomes available, even if the completion is not finished.
	StreamOptions    *struct {
		IncludeUsage bool `json:"include_usage,omitempty"`
	} `json:"stream_options,omitempty"`
	Temperature int           `json:"temperature,omitempty"` // Number between 0 and 1 that controls randomness of the output.
	TopP        int           `json:"top_p,omitempty"`       // Number between 0 and 1 that controls the cumulative probability of the output.
	Tools       []interface{} `json:"tools,omitempty"`
	ToolChoice  interface{}   `json:"tool_choice,omitempty"` // Auto but can be used to force to used a tools
	// ParallelToolCalls bool      `json:"parallel_tool_calls"`
	Messages []ChatCompletionMessageParam `json:"messages"`
}

type ChatCompletionMessageParam struct {
	// The role of the messages author, in this case `developer`.
	Role         string      `json:"role,required"`
	Audio        interface{} `json:"audio"`
	Content      interface{} `json:"content"`
	FunctionCall interface{} `json:"function_call"`
	// An optional name for the participant. Provides the model information to
	// differentiate between participants of the same role.
	Name string `json:"name"`
	// The refusal message by the assistant.
	Refusal string `json:"refusal"`
	// Tool call that this message is responding to.
	ToolCallID string      `json:"tool_call_id"`
	ToolCalls  interface{} `json:"tool_calls"`
}
