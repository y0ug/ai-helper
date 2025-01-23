package coder

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/y0ug/ai-helper/pkg/llmhaven/chat"
	"github.com/y0ug/ai-helper/pkg/llmhaven/http/streaming"
)

type StreamProcessor struct {
	writer io.Writer
	logger *slog.Logger
}

func NewStreamProcessor(writer io.Writer, logger *slog.Logger) *StreamProcessor {
	return &StreamProcessor{
		writer: writer,
		logger: logger,
	}
}

func (sp *StreamProcessor) HasWriter() bool {
	return sp.writer != nil
}

func (sp *StreamProcessor) ProcessStream(
	ctx context.Context,
	stream streaming.Streamer[chat.EventStream],
) (*chat.ChatResponse, error) {
	eventCh := make(chan chat.EventStream)
	var chatResponse *chat.ChatResponse

	go func() {
		if err := chat.StreamChatMessageToChannel(ctx, stream, eventCh); err != nil {
			if err != context.Canceled {
				sp.logger.Error("Error consuming stream", "error", err)
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case event, ok := <-eventCh:
			if !ok {
				return chatResponse, nil
			}

			switch event.Type {
			case "text_delta":
				if sp.writer != nil {
					fmt.Fprintf(sp.writer, "%v", event.Delta)
				}
			case "message_stop":
				chatResponse = event.Message
			case "error":
				chatResponse = event.Message
				return chatResponse, fmt.Errorf("error in stream: %v", event.Message)
			}
		}
	}
}
