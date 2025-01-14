package ai

// Request represents an AI generation request
type Request struct {
	Model    string    `json:"model"`
	System   string    `json:"system,omitempty"`
	Messages []Message `json:"messages"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func NewUserMessage(content string) *Message {
	return &Message{
		Role:    "user",
		Content: content,
	}
}

// Response represents an AI generation response
type Response struct {
	Content      string
	InputTokens  int
	OutputTokens int
	CachedTokens int
	Cost         *float64
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
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
		CompletionTokensDetails struct {
			ReasoningTokens          int `json:"reasoning_tokens"`
			AcceptedPredictionTokens int `json:"accepted_prediction_tokens"`
			RejectedPredictionTokens int `json:"rejected_prediction_tokens"`
		} `json:"completion_tokens_details"`
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
