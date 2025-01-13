package llmclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/y0ug/ai-helper/pkg/mcpclient"
)

// BaseProvider encapsulates common HTTP client functionalities.
type BaseProvider struct {
	apiKey   string
	client   *http.Client
	baseUrl  string
	model    *Model
	settings AIModelSettings
	logger   zerolog.Logger
}

// Type from MCP server protocol
type AITools struct {
	Description *string     `json:"description,omitempty"`
	InputSchema interface{} `json:"input_schema,omitempty"`
	Name        string      `json:"name"`
}

type AIResponse interface {
	GetChoice() AIChoice
	GetUsage() AIUsage
}

type AIUsage interface {
	GetInputTokens() int
	GetOutputTokens() int
	GetCachedTokens() int
	GetCost() float64
	SetCost(float64)
}

type AIChoice interface {
	GetMessage() AIMessage
	GetFinishReason() string
}

type AIMessage interface {
	GetRole() string
	GetContents() []*AIContent
	GetContent() *AIContent
}

type AIModelSettings interface {
	SetMaxTokens(int)
	SetTools([]AITools)
	SetStream(bool)
	SetModel(string)
}

type BaseMessage struct {
	Role    string       `json:"role"`
	Content []*AIContent `json:"content"`
}

func (m BaseMessage) GetRole() string {
	return m.Role
}

func (m BaseMessage) GetContents() []*AIContent {
	return m.Content
}

func (m BaseMessage) GetContent() *AIContent {
	if len(m.Content) != 0 {
		return m.Content[0]
	}
	return nil
}

func NewBaseMessage(role string, content ...*AIContent) BaseMessage {
	return BaseMessage{
		Role:    role,
		Content: content,
	}
}

func NewBaseMessageText(role string, text string) BaseMessage {
	return BaseMessage{
		Role: role,
		Content: []*AIContent{
			NewTextContent(text),
		},
	}
}

func ToAITools(tools []mcpclient.Tool) []AITools {
	aiTools := make([]AITools, len(tools))
	for i, tool := range tools {
		aiTools[i] = AITools{
			Description: tool.Description,
			InputSchema: tool.InputSchema,
			Name:        tool.Name,
		}
	}
	return aiTools
}

// NewBaseProvider initializes a new BaseProvider.
func NewBaseProvider(
	model *Model,
	apiKey string,
	client *http.Client,
	url string,
	logger *zerolog.Logger,
) *BaseProvider {
	if client == nil {
		client = &http.Client{}
	}

	var log zerolog.Logger
	if logger != nil {
		log = *logger
	} else {
		log = zerolog.Nop()
	}

	base := &BaseProvider{
		apiKey:  apiKey,
		client:  client,
		model:   model,
		baseUrl: url,
		logger:  log,
	}

	// base.SetModel(model)
	return base
}

// makeRequest sends an HTTP request with the given parameters, serializes the request body,
// and deserializes the response into respBody.
func (bp *BaseProvider) makeRequest(
	method, url string,
	headers map[string]string,
	reqBody interface{},
	respBody interface{},
) error {
	var buf io.Reader
	if reqBody != nil {
		jsonData, err := json.MarshalIndent(reqBody, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		buf = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	// Add additional headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if buf != nil {
		bp.logger.Debug().
			Str("method", method).
			Str("url", url).
			RawJSON("request_body", buf.(*bytes.Buffer).Bytes()).
			Msg("sending request")
	}

	resp, err := bp.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	bp.logger.Debug().
		Int("status_code", resp.StatusCode).
		RawJSON("response_body", responseBody).
		Msg("received response")

	if resp.StatusCode != http.StatusOK {
		return NewAPIError(resp.StatusCode, string(responseBody))
	}

	if respBody != nil {
		if err := json.Unmarshal(responseBody, respBody); err != nil {
			return fmt.Errorf("failed to unmarshal response body: %w", err)
		}
	}

	return nil
}

// setAuthorizationHeader sets the Authorization header if the API key is provided.
func (bp *BaseProvider) setAuthorizationHeader(headers map[string]string) {
	if bp.apiKey != "" {
		headers["Authorization"] = "Bearer " + bp.apiKey
	}
}
