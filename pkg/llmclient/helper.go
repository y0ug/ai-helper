package llmclient

import (
	"context"
	"encoding/json"
	"log"

	"github.com/y0ug/ai-helper/pkg/llmclient/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/deepseek"
	"github.com/y0ug/ai-helper/pkg/llmclient/gemini"
	"github.com/y0ug/ai-helper/pkg/llmclient/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/requestoption"
)

// WithModel sets the model for BaseChatMessageNewParams
func WithModel(model string) func(*common.BaseChatMessageNewParams) {
	return func(p *common.BaseChatMessageNewParams) {
		p.Model = model
	}
}

// WithMaxTokens sets the max tokens for BaseChatMessageNewParams
func WithMaxTokens(tokens int) func(*common.BaseChatMessageNewParams) {
	return func(p *common.BaseChatMessageNewParams) {
		p.MaxTokens = tokens
	}
}

// WithTemperature sets the temperature for BaseChatMessageNewParams
func WithTemperature(temp float64) func(*common.BaseChatMessageNewParams) {
	return func(p *common.BaseChatMessageNewParams) {
		p.Temperature = temp
	}
}

// WithMessages sets the messages for BaseChatMessageNewParams
func WithMessages(
	messages ...*common.BaseChatMessageParams,
) func(*common.BaseChatMessageNewParams) {
	return func(p *common.BaseChatMessageNewParams) {
		p.Messages = messages
	}
}

// WithTools sets the tools/functions for BaseChatMessageNewParams
func WithTools(tools ...common.Tool) func(*common.BaseChatMessageNewParams) {
	return func(p *common.BaseChatMessageNewParams) {
		p.Tools = tools
	}
}

// NewUserMessage creates a new user message
func NewUserMessage(text string) *common.BaseChatMessageParams {
	return &common.BaseChatMessageParams{
		Role: "user",
		Content: []*common.AIContent{
			common.NewTextContent(
				text,
			),
		},
	}
}

func NewUserMessageContent(content ...*common.AIContent) *common.BaseChatMessageParams {
	return &common.BaseChatMessageParams{
		Role:    "user",
		Content: content,
	}
}

// NewSystemMessage creates a new system message
func NewSystemMessage(text string) *common.BaseChatMessageParams {
	return &common.BaseChatMessageParams{
		Role: "system",
		Content: []*common.AIContent{
			common.NewTextContent(
				text,
			),
		},
	}
}

// NewMessagesParams creates a new slice of message parameters
func NewMessagesParams(msg ...*common.BaseChatMessageParams) []*common.BaseChatMessageParams {
	return msg
}

// NewChatParams creates a new BaseChatMessageNewParams with the given options
func NewChatParams(
	opts ...func(*common.BaseChatMessageNewParams),
) *common.BaseChatMessageNewParams {
	params := &common.BaseChatMessageNewParams{}
	for _, opt := range opts {
		opt(params)
	}
	return params
}

func NewDeepSeekProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &DeepseekProvider{
		client: deepseek.NewClient(opts...),
	}
}

func NewAnthropicProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &AnthropicProvider{
		client: anthropic.NewClient(opts...),
	}
}

func NewOpenAIProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &OpenAIProvider{
		client: openai.NewClient(opts...),
	}
}

func NewOpenRouterProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &GeminiProvider{
		&OpenAIProvider{
			client: gemini.NewClient(opts...).Client,
		},
	}
}

func NewGeminiProvider(opts ...requestoption.RequestOption) common.LLMProvider {
	return &GeminiProvider{
		&OpenAIProvider{
			client: gemini.NewClient().Client,
		},
	}
}

func NewProviderByModel(
	modelName string,
	infoProvider *InfoProviders,
	requestOpts ...requestoption.RequestOption,
) (common.LLMProvider, *Model) {
	model, err := ParseModel(modelName, infoProvider)
	if err != nil {
		return nil, nil
	}

	switch model.Provider {
	case "anthropic":
		return NewAnthropicProvider(requestOpts...), model
	case "openrouter":
		return NewOpenRouterProvider(requestOpts...), model
	case "openai":
		return NewOpenAIProvider(requestOpts...), model
	case "gemini":
		return NewGeminiProvider(requestOpts...), model
	case "deepseek":
		return NewDeepSeekProvider(requestOpts...), model
	default:
		return nil, model
	}
}

func ConsumeStream(
	ctx context.Context,
	stream common.Streamer[common.LLMStreamEvent],
	ch chan<- StreamEvent,
) error {
	defer close(ch)

	var msg anthropic.Message
	var cc openai.ChatCompletion
	for stream.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			event := stream.Current()
			switch event.Provider {
			case "anthropic":
				var anthropicEvent anthropic.MessageStreamEvent
				if err := json.Unmarshal(event.Data, &anthropicEvent); err != nil {
					return err
				}
				ch <- handleAnthropicEvent(&msg, &anthropicEvent)
			case "openai":
				var openaiEvent openai.ChatCompletionChunk
				if err := json.Unmarshal(event.Data, &openaiEvent); err != nil {
					return err
				}
				ch <- handleOpenAIEvent(&cc, &openaiEvent)
			default:
				log.Printf("Unknown provider: %s", event.Provider)
			}
		}
	}
	if err := stream.Err(); err != nil {
		return err
	}

	return nil
}

type StreamEvent struct {
	Type    string // text_delta, message
	Delta   interface{}
	Message *common.BaseChatMessage
}

// Return if content type text change and we return the final message
func handleAnthropicEvent(am *anthropic.Message, evt *anthropic.MessageStreamEvent) StreamEvent {
	am.Accumulate(*evt)
	svt := StreamEvent{}

	svt.Type = evt.Type

	switch evt.Type {
	case "message_start":
	case "content_block_start":
	case "content_block_delta":
		var delta common.AIContent
		err := json.Unmarshal(evt.Delta, &delta)
		if err != nil {
			return svt
		}

		switch delta.Type {
		case "text_delta":
			svt.Delta = delta.Text
			svt.Type = "text_delta"
		case "input_json_delta":
			// return delta.PartialJson
		}
	case "content_block_stop":
	case "message_delta":
		// fmt.Printf("%v\n", evt.ContentBlock)
		// fmt.Printf("Content: %v\n", evt.Delta)
	case "message_stop":
		svt.Message = AnthropicMessageToChatMessage(am)
	}

	return svt
}

func handleOpenAIEvent(cc *openai.ChatCompletion, chunk *openai.ChatCompletionChunk) StreamEvent {
	cc.Accumulate(*chunk)
	svt := StreamEvent{}
	svt.Message = OpenaiChatCompletionToChatMessage(cc)

	// Deepseek return some choices so we check if we have some usage
	if chunk.Usage.CompletionTokens != 0 {
		svt.Type = "message_stop"
		return svt
	}

	// Last openai chunk as 0 choices
	if len(chunk.Choices) == 0 {
		svt.Type = "message_stop"
		return svt
		// Last chunk
	}

	choiceDelta := chunk.Choices[0]
	svt.Delta = choiceDelta.Delta.Content
	svt.Type = "text_delta"
	return svt
}
