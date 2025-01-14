package llmclient

import (
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

func ConsumeStream(stream common.Streamer[common.LLMStreamEvent], ch chan<- string) error {
	defer close(ch)

	for stream.Next() {
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
