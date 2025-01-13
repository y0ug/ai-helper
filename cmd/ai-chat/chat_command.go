package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/y0ug/ai-helper/internal/ai"
	"github.com/reeflective/console"
)

func makeChatCommands(app *console.Console) console.Commands {
	return func() *cobra.Command {
		rootCmd := &cobra.Command{
			Use:     "chat",
			Short:   "Start an interactive chat session with AI",
			GroupID: "core",
			RunE: func(cmd *cobra.Command, args []string) error {
				agent, err := ai.NewAgent()
				if err != nil {
					return fmt.Errorf("failed to create AI agent: %w", err)
				}

				menu := app.ActiveMenu()
				
				fmt.Println("Interactive chat mode. Commands:")
				fmt.Println("  /exit, /quit - End session")
				fmt.Println("  /reset       - Clear current conversation") 
				fmt.Println("  /history     - Show chat history")
				fmt.Println("  /sessions    - List active sessions")
				fmt.Println("  /resume ID   - Resume session by ID")
				fmt.Printf("\nSession ID: %s\n", agent.ID)

				stats := SessionStats{}

				for {
					input := menu.ReadLine()
					input = strings.TrimSpace(input)

					if strings.HasPrefix(input, "/") {
						if err := handleChatCommand(input, agent, app); err != nil {
							fmt.Printf("Error: %v\n", err)
							continue
						}
						if input == "/exit" || input == "/quit" {
							return agent.Save()
						}
						continue
					}

					if input == "" {
						continue
					}

					agent.AddMessage("user", input)

					_, responses, err := agent.SendRequest()
					if err != nil {
						fmt.Printf("Error: %v\n", err)
						continue
					}

					for _, resp := range responses {
						fmt.Printf("\n%s\n", resp.GetChoice().GetMessage().GetContent())
						usage := resp.GetUsage()
						stats.SentTokens += usage.GetInputTokens()
						stats.ReceivedTokens += usage.GetOutputTokens()
						stats.MessageCost = usage.GetCost()
						stats.TotalCost += usage.GetCost()
					}

					fmt.Printf("\nModel %s | Tokens: %d sent (%d cached), %d received\n",
						agent.Model.Name,
						stats.SentTokens,
						stats.CacheHitTokens,
						stats.ReceivedTokens)
					fmt.Printf("Cost: $%.4f message, $%.4f session.\n",
						stats.MessageCost,
						stats.TotalCost)
				}
			},
		}

		return rootCmd
	}
}

type SessionStats struct {
	SentTokens       int
	CacheWriteTokens int
	CacheHitTokens   int
	ReceivedTokens   int
	MessageCost      float64
	TotalCost        float64
}

func handleChatCommand(cmd string, agent *ai.Agent, app *console.Console) error {
	parts := strings.Fields(cmd)
	switch parts[0] {
	case "/exit", "/quit":
		return nil
	case "/reset":
		agent.Messages = nil
		fmt.Println("Conversation cleared.")
	case "/history":
		fmt.Println("\nChat History:")
		for _, h := range agent.Messages {
			if h.GetRole() == "user" {
				fmt.Printf("> ")
			}
			fmt.Printf("%s\n", h.GetContent())
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
		newAgent, err := ai.LoadAgent(sessionID, agent.Model)
		if err != nil {
			return fmt.Errorf("session not found: %w", err)
		}
		*agent = *newAgent
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
	return nil
}
