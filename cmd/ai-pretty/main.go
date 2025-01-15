package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/invopop/jsonschema"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2/requestoption"
)

func StrToPtr(s string) *string {
	return &s
}

type GetWeatherInput struct {
	Location string `json:"location" jsonschema_description:"The location to look up."`
}

var GetWeatherInputSchema = GenerateSchema[GetWeatherInput]()

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	return reflector.Reflect(v)
}

func GetWeather(location string) string {
	return "Sunny"
}

func main() {
	// const model = "claude-3-5-sonnet-20241022"
	const model = "gpt-4o-mini"
	requestOpts := []requestoption.RequestOption{
		// requestoption.WithMiddleware(middleware.LoggingMiddleware()),
	}
	provider, _ := llmclient.NewProviderByModel(model, nil, requestOpts...)

	ctx := context.Background()
	params := llmclient.NewChatParams(
		llmclient.WithModel(model),
		llmclient.WithMaxTokens(1024),
		llmclient.WithTemperature(0),
		llmclient.WithMessages(
			llmclient.NewUserMessage(

				// Can you write an Hello World in C?
				"What the weather at Paris ?",
				// "Write a 500 word essai about Golang and put a some code block in the middle",
			),
		),
		llmclient.WithTools(common.Tool{
			Name:        "get_weather",
			Description: StrToPtr("Get the current weather"),
			InputSchema: GetWeatherInputSchema,
		},
		),
	)

	Exchange(ctx, provider, *params)
}

func Exchange(
	ctx context.Context,
	provider common.LLMProvider,
	params common.BaseChatMessageNewParams,
) (*common.BaseChatMessage, error) {
	var msg *common.BaseChatMessage
	var err error
	for {

		stream := provider.Stream(ctx, params)

		eventCh := make(chan llmclient.StreamEvent)

		// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
		go func() {
			// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
			if err := llmclient.ConsumeStream(ctx, stream, eventCh); err != nil {
				if err != context.Canceled {
					log.Printf("Error consuming stream: %v", err)
				}
			}
		}()

		h := highlighter.NewHighlighter(os.Stdout)
		msg, err = processStream(ctx, h, eventCh)
		if err != nil {
			log.Printf("Error processing stream: %v", err)
			return nil, nil
		}

		if msg != nil {
			fmt.Printf("\nUsage: %d %d\n", msg.Usage.InputTokens, msg.Usage.OutputTokens)
		}

		params.Messages = append(params.Messages, msg.ToMessageParams())
		choice := msg.Choice[0]
		toolResults := make([]*common.AIContent, 0)
		for _, content := range choice.Content {
			if content.Type == "tool_use" {
				log.Printf("execution: %s with \"%s\"", content.Name, string(content.Input))
				switch content.Name {
				case "get_weather":
					input := GetWeatherInput{}
					err := json.Unmarshal([]byte(content.Input), &input)
					// fmt.Println(content.InputJson)
					if err != nil {
						panic(err)
					}
					response := GetWeather(input.Location)

					b, err := json.Marshal(response)
					if err != nil {
						panic(err)
					}
					toolResults = append(
						toolResults,
						common.NewToolResultContent(content.ID, string(b)),
					)
				}
			}
		}
		if len(toolResults) == 0 {
			break
		}
		params.Messages = append(params.Messages, llmclient.NewUserMessageContent(toolResults...))
	}
	return msg, nil
}

func processStream(
	ctx context.Context,
	w io.Writer,
	ch <-chan llmclient.StreamEvent,
) (*common.BaseChatMessage, error) {
	var cm *common.BaseChatMessage
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case set, ok := <-ch:
			if !ok {
				return cm, nil
			}
			if set.Type == "text_delta" {
				fmt.Fprintf(w, "%v", set.Delta)
			}
			if set.Type == "message_stop" {
				cm = set.Message
			}
		}
	}
}
