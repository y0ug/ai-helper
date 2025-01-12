package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// BaseProvider encapsulates common HTTP client functionalities.
type BaseProvider struct {
	apiKey   string
	client   *http.Client
	baseUrl  string
	model    *Model
	settings AIModelSettings
}

type AIToolFunction struct {
	Name        string      `json:"name"`
	Description *string     `json:"description",omitempty`
	Parameters  interface{} `json:"parameters"`
}

type AITools struct {
	Type     string          `json:"type"`
	Function *AIToolFunction `json:"function",omitempty`
}

type AIResponse interface {
	GetChoice() AIChoice
	GetUsage() AIUsage
}

type AIUsage interface {
	GetInputTokens() int
	GetOutputTokens() int
	GetCachedTokens() int
}

type AIChoice interface {
	GetMessage() AIMessage
	GetFinishReason() string
}

type AIMessage interface {
	GetRole() string
	GetContents() []AIContent
	GetContent() AIContent
}

type AIModelSettings interface {
    SetMaxTokens(int)
    SetTools([]AITools)
    SetStream(bool)
    SetModel(string)
}

type BaseMessage struct {
    Role    string      `json:"role"`
    Content []AIContent `json:"content"`
}

func (m BaseMessage) GetRole() string {
	return m.Role
}

func (m BaseMessage) GetContents() []AIContent {
	return m.Content
}

func (m BaseMessage) GetContent() AIContent {
	return m.Content[0]
}

// NewBaseProvider initializes a new BaseProvider.
func NewBaseProvider(
	model *Model,
	apiKey string,
	client *http.Client,
	url string,
) *BaseProvider {
	if client == nil {
		client = &http.Client{}
	}

	base := &BaseProvider{
		apiKey:  apiKey,
		client:  client,
		model:   model,
		baseUrl: url,
	}

	// base.SetModel(model)
	return base
}

// func (bp *BaseProvider) SetModel(model *Model) {
// 	bp.model = model
// 	if bp.settings != nil {
// 		bp.settings.SetModel(model.Name)
// 	}
// }
//
// func (bp *BaseProvider) SetMaxTokens(maxTokens int) {
// 	if bp.settings != nil {
// 		bp.settings.SetMaxTokens(maxTokens)
// 	}
// }
//
// func (bp *BaseProvider) SetTools(tools []AITools) {
// 	if bp.settings != nil {
// 		bp.settings.SetTools(tools)
// 	}
// }
//
// func (bp *BaseProvider) SetStream(stream bool) {
// 	if bp.settings != nil {
// 		bp.settings.SetStream(stream)
// 	}
// }

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

	// log.Printf(
	// 	"Sending %s request to %s with body: %s",
	// 	method,
	// 	url,
	// 	string(bytes.TrimSpace(buf.(*bytes.Buffer).Bytes())),
	// )

	resp, err := bp.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	log.Printf("Received response with status %d: %s", resp.StatusCode, string(responseBody))

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
