package tokenizer

import (
	"context"
	"fmt"
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
	testCases := []struct {
		name        string
		model       string
		messages    []*chat.ChatMessage
		tools       []chat.Tool
		description string
	}{
		{
			name:  "SimpleUserMessage",
			model: "gpt-4o",
			messages: []*chat.ChatMessage{
				chat.NewUserMessage("Hello! How's the weather today?"),
			},
			tools:       nil,
			description: "Simple user message without tools",
		},
		{
			name:  "WithTools",
			model: "gpt-4o",
			messages: []*chat.ChatMessage{
				chat.NewUserMessage(
					"What's the coordinates of 13 calade st come? What the weather in Paris?",
				),
			},
			tools:       genTools(),
			description: "With tools",
		},
		{
			name:  "WithSystemMessage",
			model: "gpt-4o",
			messages: []*chat.ChatMessage{
				chat.NewSystemMessage("You're a comedian."),
				chat.NewUserMessage("Tell me a joke!"),
			},
			description: "System message",
		},
		{
			name:  "WithToolsSystemMessage",
			model: "gpt-4o",
			messages: []*chat.ChatMessage{
				chat.NewSystemMessage("You are are a weather forecaster."),
				chat.NewUserMessage("What the weather in paris!"),
			},
			tools:       genTools(),
			description: "System message with tools",
		},
		{
			name:  "MultiTurnConversation",
			model: "gpt-4o",
			messages: []*chat.ChatMessage{
				chat.NewUserMessage("What's the capital of France?"),
				chat.NewMessage(
					"assistant",
					chat.NewTextContent("The capital of France is Paris."),
				),
				chat.NewUserMessage("What's the population there?"),
			},
			tools:       nil,
			description: "Multi-turn conversation history",
		},
		// Add more test cases for different models
		{
			name:  "GPT3.5Turbo",
			model: "gpt-3.5-turbo",
			messages: []*chat.ChatMessage{
				chat.NewUserMessage("Explain quantum computing in simple terms"),
			},
			tools:       nil,
			description: "Different model (gpt-3.5-turbo)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate model encoding
			encoding := GetEncoding(tc.model)
			if encoding == "" {
				t.Skipf("Skipping %s: encoding not found for model %s", tc.description, tc.model)
				return
			}

			// Initialize tokenizer
			tkm, err := tiktoken.GetEncoding(encoding)
			if err != nil {
				t.Fatalf("Failed to get encoding: %v", err)
			}

			// Calculate local token counts
			localInputTokens := CountRequest(tkm, tc.messages, tc.tools, false)

			// Make API call
			resp, err := ChatCompletion(
				context.Background(),
				"openai",
				tc.model,
				tc.messages,
				tc.tools,
			)
			if err != nil {
				t.Fatalf("API call failed: %v", err)
			}

			// Calculate output tokens from response
			respMsg := resp.ToMessageParams()
			localOutputTokens, _, _ := CountMessage(tkm, true, respMsg)

			totalMsgs := append(tc.messages, respMsg)
			totalTokens := CountRequest(tkm, totalMsgs, tc.tools, true)

			// Log results
			t.Logf("\n=== Test Case: %s ===", tc.description)
			t.Logf("Model: %s", tc.model)
			t.Logf("Local vs API Input Tokens: %d vs %d", localInputTokens, resp.Usage.InputTokens)
			t.Logf(
				"Local vs API Output Tokens: %d vs %d",
				localOutputTokens,
				resp.Usage.OutputTokens,
			)

			t.Logf(
				"Local vs API Total Tokens: %d vs %d",
				localInputTokens+localOutputTokens,
				resp.Usage.InputTokens+resp.Usage.OutputTokens+resp.Usage.InputCachedTokens,
			)

			t.Logf(
				"TotalTokens vs API Total Tokens: %d vs %d",
				totalTokens,
				resp.Usage.InputTokens+resp.Usage.OutputTokens+resp.Usage.InputCachedTokens,
			)

			t.Logf(
				"Local vs TotalTokens Total Tokens: %d vs %d",
				localInputTokens+localOutputTokens,
				totalTokens,
			)

			// Add validation thresholds (adjust based on expected variance)
			if abs(totalTokens-localOutputTokens-localInputTokens) > 2 {
				t.Errorf(
					"Total token mismatch exceeds threshold: %d vs %d",
					totalTokens,
					localOutputTokens+localInputTokens,
				)
			}
			if abs(localInputTokens-resp.Usage.InputTokens) > 2 {
				t.Errorf(
					"Input token mismatch exceeds threshold: %d vs %d",
					localInputTokens,
					resp.Usage.InputTokens,
				)
			}

			if abs(localOutputTokens-resp.Usage.OutputTokens) > 2 {
				t.Errorf(
					"Output token mismatch exceeds threshold: %d vs %d",
					localOutputTokens,
					resp.Usage.OutputTokens,
				)
			}
		})
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
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
