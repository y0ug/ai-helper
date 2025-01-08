package coder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/y0ug/ai-helper/internal/ai"
	"go.uber.org/mock/gomock"
)

func TestCoder_Integration(t *testing.T) {
	// Setup
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock client
	mockClient := ai.NewMockAIClient(ctrl)

	// Create test agent with mock client
	agent := &ai.Agent{
		Client:   mockClient,
		Messages: []ai.Message{},
	}

	// Test files
	files := map[string]string{
		"test.go": "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}",
	}

	// Expected responses from AI
	mockClient.EXPECT().
		GenerateWithMessages(gomock.Any(), gomock.Any()).
		Return(ai.Response{Content: "Understood the request"}, nil)

	mockClient.EXPECT().
		GenerateWithMessages(gomock.Any(), gomock.Any()).
		Return(ai.Response{Content: `test.go
` + "```" + `go
<<<<<<< SEARCH
func main() {
	println("hello")
}
 =======
 func main() {
     println("hello world")
 }
 >>>>>>> REPLACE
 ` + "```"}, nil)

	// Create coder instance
	coder := New(agent)

	// Test request
	resp, err := coder.RequestChange(context.Background(), "Update the print message", files)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Contains(t, resp.ModifiedFiles["test.go"], "hello world")
}
