package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/llmagent"
	"github.com/y0ug/ai-helper/internal/middleware"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
	"github.com/y0ug/ai-helper/pkg/llmclient/modelinfo"
)

var mcpConfig = `
mcpServers:
  sequentialthinking:
    command: "docker"
    args:
      - run
      - --rm
      - -i
      - mcp/sequentialthinking
  brave-search:
    command: "docker"
    args:
      - run
      - --rm
      - -i
      - -e
      - BRAVE_API_KEY
      - mcp/brave-search
  time:
    command: "docker"
    args:
      - run
      - --rm
      - -i
      - mcp/time
`

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// const model = "gpt-4o"
	const model = "gpt-4o-mini"
	ctx := context.Background()
	// const model = "gpt-4o"
	requestOpts := []options.RequestOption{
		// requestoption.WithMiddleware(middleware.LoggingMiddleware()),
		options.WithMiddleware(middleware.TimeitMiddleware(logger)),
	}

	loader := config.NewLoader()
	cfg, err := loader.LoadData([]byte(mcpConfig), "yml")
	if err != nil {
		fmt.Println(err)
		return
	}

	cachePath := "/tmp"
	modelInfoProvider, err := modelinfo.New(filepath.Join(cachePath, "modelinfo.json"))
	if err != nil {
		logger.Error("failed to create model info provider", "error", err)
		return
	}

	chatParams := chat.NewChatParams(
		chat.WithModel(model),
		chat.WithMaxTokens(100),
	)

	agent, err := llmagent.New(
		"test",
		logger,
		chatParams,
		modelInfoProvider,
		&cfg.MCPServers,
		requestOpts...)
	if err != nil {
		fmt.Println(err)
		return
	}

	agent.StartMCP(ctx)

	h := highlighter.NewHighlighter(os.Stdout)
	agent.AddMessage(chat.NewMessage("user",
		chat.NewTextContent("What time is it at New York?")))
	// chat.NewTextContent("What the weather at Paris?")))
	_, cost, err := agent.Do(ctx, h)
	if err != nil {
		fmt.Println(err)
		return
	}

	logger.Info("cost", "value", cost)
	agent.StopMCP()
}
