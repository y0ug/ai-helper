package coder

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/y0ug/ai-helper/internal/ai"
)

func TestCoder_ClaudeIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "1" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=1 to run")
	}

	// Ensure API key is set
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	require.NotEmpty(t, apiKey, "ANTHROPIC_API_KEY environment variable must be set")

	// Create real Claude client
	client := ai.NewClaudeClient(apiKey, "claude-3-sonnet-20241022")

	// Create test agent with real client
	agent := &ai.Agent{
		Client:   client,
		Messages: []ai.Message{},
	}

	// Test files
	files := map[string]string{
		"example.go": `package main

func main() {
	// TODO: Implement proper greeting
	println("hi")
}`,
	}

	// Create coder instance
	coder := New(agent)
	coder.SetTemplateData(files)

	// Test request
	resp, err := coder.RequestChange(context.Background(), 
		"Update the greeting to print 'Hello, World!' with proper formatting", 
		files)

	// Assertions
	require.NoError(t, err)
	require.NotNil(t, resp)
	
	// Verify the changes
	assert.Contains(t, resp.ModifiedFiles["example.go"], "Hello, World!")
	assert.Contains(t, resp.Analysis, "greeting")
	assert.NotEmpty(t, resp.Changes)
}
