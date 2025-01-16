package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/y0ug/ai-helper/internal/middleware"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient"
	"github.com/y0ug/ai-helper/pkg/llmclient/common"
	"github.com/y0ug/ai-helper/pkg/llmclient/requestoption"
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
	conditions := []string{
		"Sunny",
		"Cloudy",
		"Rainy",
		"Partly cloudy",
		"Thunderstorm",
		"Windy",
		"Snowy",
		"Foggy",
	}
	rand.Seed(time.Now().UnixNano())
	return conditions[rand.Intn(len(conditions))]
}

func main() {
	const model = "claude-3-5-sonnet-20241022"
	// const model = "gpt-4o"
	requestOpts := []requestoption.RequestOption{
		// requestoption.WithMiddleware(middleware.LoggingMiddleware()),
		requestoption.WithMiddleware(middleware.TimeitMiddleware()),
	}
	modelInfoProvider, _ := llmclient.NewModelInfoProvider("")
	provider, _ := llmclient.NewProviderByModel(model, modelInfoProvider, requestOpts...)

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
	// choices := 3
	// params.N = &choices
	_, err := HandleLLMConversation(ctx, provider, *params)
	if err != nil {
		fmt.Println(err)
	}
	// for _, choice := range msg.Choice {
	// 	fmt.Println(choice.Content[0])
	// }
}

func HandleLLMConversation(
	ctx context.Context,
	provider llmclient.LLMProvider,
	params common.ChatMessageNewParams,
) (*common.ChatMessage, error) {
	var msg *common.ChatMessage
	for {

		stream, err := provider.Stream(ctx, params)
		if err != nil {
			log.Printf("Error streaming: %v", err)
			return nil, err
		}

		eventCh := make(chan common.EventStream)

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

		if msg == nil {
			log.Printf("No message returned")
			return nil, nil
		}
		fmt.Printf("\nUsage: %d %d\n", msg.Usage.InputTokens, msg.Usage.OutputTokens)

		params.Messages = append(params.Messages, msg.ToMessageParams())
		toolResults := make([]*common.AIContent, 0)
		// for _, choice := range msg.Choice {
		choice := msg.Choice[0]
		for _, content := range choice.Content {
			if content.Type == "tool_use" {
				log.Printf(
					"%s execution: %s with \"%s\"",
					content.ID,
					content.Name,
					string(content.Input),
				)
				switch content.Name {
				case "get_weather":
					input := GetWeatherInput{}
					err := json.Unmarshal([]byte(content.Input), &input)
					// fmt.Println(content.InputJson)
					if err != nil {
						panic(err)
					}
					response := GetWeather(input.Location)
					fmt.Println(response)

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
		// }
		if len(toolResults) == 0 {
			break
		}

		// if params.N != nil {
		// 	*params.N = 1
		// }

		params.Messages = append(params.Messages, llmclient.NewUserMessageContent(toolResults...))
	}
	return msg, nil
}

func processStream(
	ctx context.Context,
	w io.Writer,
	ch <-chan common.EventStream,
) (*common.ChatMessage, error) {
	var cm *common.ChatMessage
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
