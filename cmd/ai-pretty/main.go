package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/y0ug/ai-helper/internal/config"
	"github.com/y0ug/ai-helper/internal/llmagent"
	"github.com/y0ug/ai-helper/internal/middleware"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
	"github.com/y0ug/ai-helper/pkg/llmclient/http/options"
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
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	logger := zerolog.New(output).Level(zerolog.DebugLevel).With().Timestamp().Logger()

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

	chatParams := chat.NewChatParams(
		chat.WithModel(model),
		chat.WithMaxTokens(100),
	)
	agent, err := llmagent.New("test", logger, chatParams, "/tmp/", &cfg.MCPServers, requestOpts...)
	if err != nil {
		fmt.Println(err)
		return
	}

	agent.StartMCP(ctx)

	h := highlighter.NewHighlighter(os.Stdout)
	agent.AddMessage(chat.NewMessage("user",
		// chat.NewTextContent("What time is it at New York?")))
		chat.NewTextContent("What the weather at Paris?")))
	_, cost, err := agent.Do(ctx, h)
	if err != nil {
		fmt.Println(err)
		return
	}

	logger.Info().Float64("cost", cost).Msgf("cost")
	agent.StopMCP()
}
