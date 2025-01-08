package chat

import (
	"bufio"
	"fmt"
	"os"
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
	agent *ai.Agent
	stats SessionStats
}

func NewChat(agent *ai.Agent) *Chat {
	return &Chat{
		agent: agent,
	}
}

func (c *Chat) Start() error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Interactive chat mode. Commands:")
	fmt.Println("  /exit, /quit - End session")
	fmt.Println("  /reset       - Clear current conversation")
	fmt.Println("  /history     - Show chat history")
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
				return c.agent.Save()
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
		resp, err := c.agent.SendRequest()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Print("\n> ")
			continue
		}

		fmt.Printf("\n%s\n", resp.Content)
		// Update session stats
		c.stats.SentTokens += resp.InputTokens
		c.stats.ReceivedTokens += resp.OutputTokens

		if resp.Cost != nil {
			c.stats.MessageCost = *resp.Cost
			c.stats.TotalCost += *resp.Cost
		}

		// Calculate cache metrics
		newCacheHits := 0
		if resp.InputTokens >= 1024 {
			// Round down to nearest 128 token increment
			newCacheHits = (resp.CachedTokens / 128) * 128
		}
		c.stats.CacheHitTokens += newCacheHits
		c.stats.CacheWriteTokens += resp.InputTokens - newCacheHits

		fmt.Printf("\nModel %s | Tokens: %d sent (%d cached), %d received\n",
			c.agent.Model.Name,
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
	case "/reset":
		c.agent.Messages = nil
		fmt.Println("Conversation cleared.")
	case "/history":
		fmt.Println("\nChat History:")
		for _, h := range c.agent.Messages {
			if h.Role == "user" {
				fmt.Printf("> ")
			}
			fmt.Printf("%s\n", h.Content)
		}

	case "/sessions":
		sessions, err := ai.ListAgents()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}
		for _, v := range sessions {
			fmt.Printf("%s\n", v)
		}

	case "/resume":
		if len(parts) != 2 {
			return fmt.Errorf("usage: /resume SESSION_ID")
		}
		sessionID := parts[1]
		newAgent, err := ai.LoadAgent(sessionID, c.agent.Model)
		if err != nil {
			return fmt.Errorf("session not found: %w", err)
		}
		c.agent = newAgent
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
	fmt.Print("\n> ")
	return nil
}
