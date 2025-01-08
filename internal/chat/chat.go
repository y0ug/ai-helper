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
	SessionID string       `json:"session_id"`
	Messages  []ai.Message `json:"messages"`
	Date      time.Time    `json:"date"`
	Model     string       `json:"model"`
}

type SessionStats struct {
	SentTokens       int
	CacheWriteTokens int
	CacheHitTokens   int
	ReceivedTokens   int
	MessageCost      float64
	TotalCost        float64
}

type Chat struct {
	client       *ai.Client
	agent        *ai.Agent
	historyFile  string
	historyCache []ChatHistory
	stats        SessionStats
}

func generateSessionID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}

func NewChat(client *ai.Client) (*Chat, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}

	aiHelperCache := filepath.Join(cacheDir, "ai-helper")
	if err := os.MkdirAll(aiHelperCache, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	historyFile := filepath.Join(aiHelperCache, "chat_history.json")
	sessionID := generateSessionID()
	
	agent := client.CreateAgent(sessionID)

	chat := &Chat{
		client:      client,
		agent:       agent,
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

func (c *Chat) SetSystemPrompt(prompt string) {
	if prompt != "" {
		c.agent.AddSystemMessage(prompt)
	}
}

func (c *Chat) SetInitialPrompt(prompt string) {
	if prompt != "" {
		c.agent.AddMessage("user", prompt)
	}
}

func (c *Chat) saveHistory() error {
	if len(c.agent.Messages) > 0 {
		history := ChatHistory{
			SessionID: c.agent.ID,
			Messages:  c.agent.Messages,
			Date:      time.Now(),
			Model:     c.agent.Model.String(),
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
	fmt.Println("  /sessions    - List active sessions")
	fmt.Println("  /resume ID   - Resume session by ID")
	fmt.Printf("\nSession ID: %s\n", c.agent.ID)
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

		// Add user message to agent
		c.agent.AddMessage("user", input)

		// Generate response using the agent
		resp, err := c.client.GenerateForAgent(c.agent, "chat")
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Print("\n> ")
			continue
		}


		fmt.Printf("\n%s\n", resp.Content)
		// Update session stats
		c.stats.SentTokens += resp.InputTokens
		c.stats.ReceivedTokens += resp.OutputTokens
		c.stats.MessageCost = resp.Cost
		c.stats.TotalCost += resp.Cost

		// Calculate cache metrics
		newCacheHits := 0
		if resp.InputTokens >= 1024 {
			// Round down to nearest 128 token increment
			newCacheHits = (resp.CachedTokens / 128) * 128
		}
		c.stats.CacheHitTokens += newCacheHits
		c.stats.CacheWriteTokens += resp.InputTokens - newCacheHits

		fmt.Printf("\nTokens: %d sent (%d cached), %d received\n",
			c.stats.SentTokens,
			c.stats.CacheHitTokens,
			c.stats.ReceivedTokens)
		fmt.Printf("Cost: $%.4f message, $%.4f session.\n",
			c.stats.MessageCost,
			c.stats.TotalCost)
		fmt.Print("\n> ")
	}
}

func (c *Chat) handleCommand(cmd string) error {
	parts := strings.Fields(cmd)
	switch parts[0] {
	case "/exit", "/quit":
		return nil
	case "/clear":
		c.agent.Messages = nil
		fmt.Println("Conversation cleared.")
	case "/history":
		fmt.Println("\nChat History:")
		for i, h := range c.historyCache {
			preview := h.Messages[0].Content
			if len(preview) > 60 {
				preview = preview[:57] + "..."
			}
			fmt.Printf(
				"%d: [%s] (Session: %s) %s\n",
				i,
				h.Date.Format("2006-01-02 15:04"),
				h.SessionID,
				preview,
			)
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
		c.agent.Messages = make([]ai.Message, len(c.historyCache[index].Messages))
		copy(c.agent.Messages, c.historyCache[index].Messages)
		fmt.Println("Loaded conversation from history.")
	case "/sessions":
		fmt.Println("\nActive Sessions:")
		sessions := make(map[string]time.Time)
		for _, h := range c.historyCache {
			if _, exists := sessions[h.SessionID]; !exists {
				sessions[h.SessionID] = h.Date
			}
		}
		for id, date := range sessions {
			fmt.Printf("%s: Last active %s\n", id, date.Format("2006-01-02 15:04"))
		}
	case "/resume":
		if len(parts) != 2 {
			return fmt.Errorf("usage: /resume SESSION_ID")
		}
		sessionID := parts[1]
		var found bool
		for _, h := range c.historyCache {
			if h.SessionID == sessionID {
				c.agent.Messages = make([]ai.Message, len(h.Messages))
				copy(c.agent.Messages, h.Messages)
				c.agent.ID = sessionID
				found = true
				fmt.Printf("Resumed session %s\n", sessionID)
				break
			}
		}
		if !found {
			return fmt.Errorf("session not found: %s", sessionID)
		}
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
	fmt.Print("\n> ")
	return nil
}
