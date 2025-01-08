package coder

import (
    "context"
    "testing"
    "github.com/yourusername/yourproject/internal/ai"
)

type mockModel struct {
    responses []string
    current   int
}

func (m *mockModel) SendMessages(messages []ai.Message) (*ai.Response, error) {
    resp := m.responses[m.current]
    m.current++
    return &ai.Response{Content: resp}, nil
}

func TestService_ProcessRequest(t *testing.T) {
    mockModel := &mockModel{
        responses: []string{
            "Analysis: Need to update the print statement",
            `test.py
` + "```" + `python
<<<<<<< SEARCH
print("hello")
