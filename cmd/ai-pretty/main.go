package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/invopop/jsonschema"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient/v2"
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
	const model = "claude-3-5-sonnet-20241022"
	// const model = "gpt-4o"
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
				`
        Can you write some basic json data about user information in code block fenced by three backticks?
        `,
				// Can you write an Hello World in C?
				// "What the weather at Paris ?",
				// "Write a 1000 word essai about Golang and put a some code block in the middle",
			),
		),
		// llmclient.WithTools(common.Tool{
		// 	Name:        "get_weather",
		// 	Description: StrToPtr("Get the current weather"),
		// 	InputSchema: GetWeatherInputSchema,
		// },
		// ),
	)

	stream := provider.Stream(ctx, *params)
	eventCh := make(chan string)

	// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
	go func() {
		// llmclient.ConsumeStreamIO(ctx, stream, os.Stdout)
		if err := llmclient.ConsumeStream(ctx, stream, eventCh); err != nil {
			if err != context.Canceled {
				log.Printf("Error consuming stream: %v", err)
			}
		}
	}()

	writer := bufio.NewWriter(os.Stdout)
	h := highlighter.NewHighlighter(writer)
	h.ProcessStream(ctx, eventCh)
	// processStream(ctx, os.Stdout, eventCh)
}

func processStream(ctx context.Context, w io.Writer, ch <-chan string) error {
	// f := bufio.NewWriter(w)
	// defer f.Flush()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case content, ok := <-ch:
			if !ok {
				break
			}
			fmt.Fprintf(w, "%s", content)
		}
	}
}
