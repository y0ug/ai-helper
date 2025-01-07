package ai

// Request represents an AI generation request
type Request struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents an AI generation response
type Response struct {
	Content      string
	InputTokens  int
	OutputTokens int
	Cost         float64
	Error        error
}

// APIResponse represents the standard response format from OpenAI/OpenRouter providers
type APIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// AnthropicResponse represents Anthropic's specific response format
type AnthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
		Type string `json:"type"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// NewRequest creates a new AI generation request
func NewRequest(prompt string) *Request {
	return &Request{
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}
}
