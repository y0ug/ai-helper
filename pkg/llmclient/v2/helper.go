package llmclient

import (
	"context"
	"encoding/json"
	"log"

	"github.com/y0ug/ai-helper/pkg/llmclient"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/anthropic"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/deepseek"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/gemini"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/openai"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
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
func WithTools(tools []common.Tool) func(*common.BaseChatMessageNewParams) {
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

func NewGeminiProvider() common.LLMProvider {
	return &GeminiProvider{
		&OpenAIProvider{
			client: gemini.NewClient().Client,
		},
	}
}

func NewProviderByModel(
	modelName string,
	infoProvider *llmclient.InfoProviders,
) (common.LLMProvider, *llmclient.Model) {
	model, err := llmclient.ParseModel(modelName, infoProvider)
	if err != nil {
		return nil, nil
	}

	switch model.Provider {
	case "anthropic":
		return NewAnthropicProvider(), model
	case "openrouter":
		return NewOpenRouterProvider(), model
	case "openai":
		return NewOpenAIProvider(), model
	case "gemini":
		return NewGeminiProvider(), model
	case "deepseek":
		return NewDeepSeekProvider(), model
	default:
		return nil, model
	}
}

func ConsumeStream(
	ctx context.Context,
	stream common.Streamer[common.LLMStreamEvent],
	ch chan<- string,
) error {
	defer close(ch)

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
				ch <- handleAnthropicEvent(anthropicEvent)
			case "openai":
				var openaiEvent openai.ChatCompletionChunk
				if err := json.Unmarshal(event.Data, &openaiEvent); err != nil {
					return err
				}
				ch <- handleOpenAIEvent(openaiEvent)
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

func handleAnthropicEvent(evt anthropic.MessageStreamEvent) string {
	// Implement handling logic

	switch evt.Type {
	case "message_start":
	case "content_block_start":
	case "content_block_delta":
		// fmt.Printf("%v\n", evt.ContentBlock)
		// fmt.Printf("Content: %v\n", evt.Delta)
		// fmt.Printf("%s", evt.Delta)
		return evt.Delta.Text
	case "content_block_stop":
	case "message_delta":
		// fmt.Printf("%v\n", evt.ContentBlock)
		// fmt.Printf("Content: %v\n", evt.Delta)
	case "message_stop":
	}

	return ""
}

func handleOpenAIEvent(evt openai.ChatCompletionChunk) string {
	// Implement handling logic
	// fmt.Printf("OpenAI Event: %+v\n", evt)
	if len(evt.Choices) == 0 {
		return ""
	}
	// print(fmt.Sprintf("### %s", evt.Choices[0].Delta.Content))
	return evt.Choices[0].Delta.Content
}
