package tokenizer

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/pkoukk/tiktoken-go"
	"github.com/y0ug/ai-helper/internal/middleware"
	"github.com/y0ug/ai-helper/pkg/llmhaven"
	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
	"github.com/y0ug/ai-helper/pkg/llmhaven/http/options"
)

type GetCoordinatesInput struct {
	Location string `json:"location" jsonschema_description:"The location to look up."`
}

var GetCoordinatesInputSchema = GenerateSchema[GetCoordinatesInput]()

type GetCoordinateResponse struct {
	Long float64 `json:"long"`
	Lat  float64 `json:"lat"`
}

func GetCoordinates(location string) GetCoordinateResponse {
	return GetCoordinateResponse{
		Long: -122.4194,
		Lat:  37.7749,
	}
}

// Get Temperature Unit

type GetTemperatureUnitInput struct {
	Country string `json:"country" jsonschema_description:"The country"`
}

var GetTemperatureUnitInputSchema = GenerateSchema[GetTemperatureUnitInput]()

func GetTemperatureUnit(country string) string {
	return "farenheit"
}

// Get Weather

type GetWeatherInput struct {
	Lat  float64 `json:"lat"  jsonschema_description:"The latitude of the location to check weather."`
	Long float64 `json:"long" jsonschema_description:"The longitude of the location to check weather."`
	Unit string  `json:"unit" jsonschema_description:"Unit for the output"`
}

var GetWeatherInputSchema = GenerateSchema[GetWeatherInput]()

type GetWeatherResponse struct {
	Unit        string  `json:"unit"`
	Temperature float64 `json:"temperature"`
}

func GetWeather(lat, long float64, unit string) GetWeatherResponse {
	return GetWeatherResponse{
		Unit:        "farenheit",
		Temperature: 122,
	}
}

func GenerateSchema[T any]() interface{} {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	return reflector.Reflect(v)
}

func ToPtr[T any](s T) *T {
	return &s
}

func genTools() []chat.Tool {
	tools := []chat.Tool{
		{
			Name: "get_coordinates",
			Description: ToPtr(
				"Accepts a place as an address, then returns the latitude and longitude coordinates.",
			),
			InputSchema: GetCoordinatesInputSchema,
		},
		{
			Name:        "get_temperature_unit",
			InputSchema: GetTemperatureUnitInputSchema,
		},
		{
			Name:        "get_weather",
			Description: ToPtr("Get the weather at a specific location"),
			InputSchema: GetWeatherInputSchema,
		},
	}
	return tools
}

func TestNumTokensFromMessages(t *testing.T) {
	modelName := "gpt-4o"

	encoding := GetEncoding(modelName)
	if encoding == "" {
		err := fmt.Errorf("encoding for model %s not found", modelName)
		t.Log(err)
		return
	}

	fmt.Printf("Model: %s Endcoding: %s", modelName, encoding)
	tkm, err := tiktoken.GetEncoding("o200k_base")
	if err != nil {
		err = fmt.Errorf("encoding for model: %v", err)
		log.Println(err)
		return
	}

	msgs := []*chat.ChatMessage{
		// chat.NewUserMessage("Hello test token! What is the weather today?"),
		chat.NewUserMessage(
			"I'm unable to provide real-time data, including current weather conditions. I recommend checking a reliable weather website or app for the most up-to-date information. If you have any other questions or need information, feel free to ask!",
		),
		// chat.NewMessage("assistant", chat.NewTextContent("The weather is sunny today.")),
	}

	t.Run("TestNumTokensMessageIntergration", func(t *testing.T) {
		tools := []chat.Tool{}
		inputTokens := CountRequest(tkm, msgs, nil, false)
		fmt.Printf(
			"Model: %s, InputTokens: %d Msgs: %d Tools: %d\n",
			modelName,
			inputTokens,
			len(msgs),
			len(tools),
		)

		resp, err := ChatCompletion(context.Background(), "openai", "gpt-4o", msgs, tools)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Printf(
			"Resp Choice: %d Contents: %d\n",
			len(resp.Choice),
			len(resp.Choice[0].Content),
		)

		respMsg := resp.ToMessageParams()
		fmt.Printf("content: %s\n", respMsg.Content[0].String())

		outputTokens := len(tkm.Encode(respMsg.Role, nil, nil))
		for _, content := range respMsg.Content {
			if content.Type == chat.ContentTypeText {
				outputTokens += len(tkm.Encode(content.Text, nil, nil))
			}
		}

		msgs = append(msgs, respMsg)
		totalTokens := CountRequest(tkm, msgs, nil, false)

		// outputTokens := CountRequest(tkm, []*chat.ChatMessage{respMsg}, nil, false)
		fmt.Printf(
			"Local: InputTokens: %d OutputToken %d totalTokens %d\n",
			inputTokens,
			outputTokens,
			totalTokens,
		)
		fmt.Printf(
			"Usage: InputTokens: %d OutputToken %d InputCachedTokens %d totalTokens %d\n",
			resp.Usage.InputTokens,
			resp.Usage.OutputTokens,
			resp.Usage.InputCachedTokens,
			resp.Usage.InputTokens+resp.Usage.OutputTokens+resp.Usage.InputCachedTokens,
		)
	})
}

func ChatCompletion(
	ctx context.Context,
	provider string,
	model string,
	msgs []*chat.ChatMessage,
	tools []chat.Tool,
) (*chat.ChatResponse, error) {
	ctxRequest, cancelFn := context.WithTimeout(ctx, 10*time.Second)
	defer cancelFn()

	requestOpts := []options.RequestOption{
		options.WithMiddleware(middleware.LoggingMiddleware()),
		// options.WithMiddleware(middleware.TimeitMiddleware(nil)),
	}

	// modelInfoProvider, _ := modelinfo.New("")

	llm, err := llmhaven.New(provider, requestOpts...)
	if err != nil {
		return nil, fmt.Errorf("Failed to create provider: %v", err)
	}

	params := chat.NewChatParams(
		chat.WithModel(model),
		chat.WithMessages(msgs...),
		chat.WithTools(tools...),
		chat.WithMaxTokens(1000),
	)
	resp, err := llm.Send(
		ctxRequest,
		*params,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to send message: %v", err)
	}

	if resp == nil {
		return nil, fmt.Errorf("Response is nil")
	}

	return resp, nil
}
