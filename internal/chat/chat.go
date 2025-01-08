package chat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/y0ug/ai-helper/internal/ai"
)

type ChatHistory struct {
	Messages []ai.Message `json:"messages"`
	Date     time.Time   `json:"date"`
	Model    string      `json:"model"`
}

type Chat struct {
	client       *ai.Client
	model        string
	messages     []ai.Message
	historyFile  string
	historyCache []ChatHistory
}

func NewChat(client *ai.Client, model string) (*Chat, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}

	// Create ai-helper cache directory if it doesn't exist
	aiHelperCache := filepath.Join(cacheDir, "ai-helper")
	if err := os.MkdirAll(aiHelperCache, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	historyFile := filepath.Join(aiHelperCache, "chat_history.json")

	chat := &Chat{
		client:      client,
		model:       model,
		historyFile: historyFile,
	}

	if err := chat.loadHistory(); err != nil {
		return nil, fmt.Errorf("failed to load history: %w", err)
	}

	return chat, nil
}

func (c *Chat) loadHistory() error {
	data, err := os.ReadFile(c.historyFile)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}

	return json.Unmarshal(data, &c.historyCache)
}

func (c *Chat) saveHistory() error {
	if len(c.messages) > 0 {
		history := ChatHistory{
			Messages: c.messages,
			Date:     time.Now(),
			Model:    c.model,
		}
		c.historyCache = append([]ChatHistory{history}, c.historyCache...)

		// Keep only last 50 conversations
		if len(c.historyCache) > 50 {
			c.historyCache = c.historyCache[:50]
		}
	}

	data, err := json.MarshalIndent(c.historyCache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.historyFile, data, 0644)
}

func (c *Chat) Start() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Interactive chat mode. Commands:")
	fmt.Println("  /exit, /quit - End session")
	fmt.Println("  /clear       - Clear current conversation")
	fmt.Println("  /history     - Show chat history")
	fmt.Println("  /load N      - Load conversation N from history")
	fmt.Print("\n> ")

	for {
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}

		input = strings.TrimSpace(input)

		// Handle commands
		if strings.HasPrefix(input, "/") {
			if err := c.handleCommand(input); err != nil {
				fmt.Printf("Error: %v\n", err)
				fmt.Print("\n> ")
				continue
			}
			if input == "/exit" || input == "/quit" {
				return c.saveHistory()
			}
			continue
		}

		if input == "" {
			fmt.Print("> ")
			continue
		}

		// Add user message
		c.messages = append(c.messages, ai.Message{
			Role:    "user",
			Content: input,
		})

		// Get response
		req := &ai.Request{
			Model:    c.model,
			Messages: c.messages,
		}

		resp, err := c.client.Generate(input, "chat")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Print("\n> ")
			continue
		}

		// Add assistant response
		c.messages = append(c.messages, ai.Message{
			Role:    "assistant",
			Content: resp.Content,
		})

		fmt.Printf("\n%s\n", resp.Content)
		fmt.Printf("\n[Tokens: %d in, %d out | Cost: $%.4f]\n",
			resp.InputTokens,
			resp.OutputTokens,
			resp.Cost)
		fmt.Print("\n> ")
	}
}

func (c *Chat) handleCommand(cmd string) error {
	parts := strings.Fields(cmd)
	switch parts[0] {
	case "/exit", "/quit":
		return nil
	case "/clear":
		c.messages = nil
		fmt.Println("Conversation cleared.")
	case "/history":
		fmt.Println("\nChat History:")
		for i, h := range c.historyCache {
			preview := h.Messages[0].Content
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}
			fmt.Printf("%d: [%s] %s\n", i, h.Date.Format("2006-01-02 15:04"), preview)
		}
	case "/load":
		if len(parts) != 2 {
			return fmt.Errorf("usage: /load N")
		}
		var index int
		if _, err := fmt.Sscanf(parts[1], "%d", &index); err != nil {
			return fmt.Errorf("invalid history index: %s", parts[1])
		}
		if index < 0 || index >= len(c.historyCache) {
			return fmt.Errorf("history index out of range")
		}
		c.messages = make([]ai.Message, len(c.historyCache[index].Messages))
		copy(c.messages, c.historyCache[index].Messages)
		fmt.Println("Loaded conversation from history.")
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
	fmt.Print("\n> ")
	return nil
}
