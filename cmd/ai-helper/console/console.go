package console

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/reeflective/console"
	"github.com/spf13/cobra"

	"github.com/y0ug/ai-helper/internal/llmagent"
	"github.com/y0ug/ai-helper/internal/version"
	"github.com/y0ug/ai-helper/pkg/highlighter"
	"github.com/y0ug/ai-helper/pkg/llmclient/chat"
)

type MyCmd struct {
	cobra.Command
}

func (c *MyCmd) Find(args []string) (*cobra.Command, []string, error) {
	fmt.Println("Finding command")
	return nil, nil, nil
}

//
// func (c *MyConsole) execute(
// 	ctx context.Context,
// 	menu *console.Menu,
// 	args []string,
// 	async bool,
// ) error {
// 	fmt.Println("Executing command")
// 	if strings.HasPrefix(args[0], "/") {
// 		return c.execute(ctx, menu, args, async)
// 	} else {
// 		args = append([]string{"msg"}, args...)
// 		return c.execute(ctx, menu, args, async)
// 	}
// }
//
// func NewMyConsole(name string) *MyConsole {
// 	return &MyConsole{
// 		Console: *console.New(name),
// 	}
// }

func StartConsole(agent *llmagent.Agent) {
	app := console.New("ai-helper")

	app.NewlineBefore = true
	app.NewlineAfter = true
	menu := app.ActiveMenu()
	menu.AddInterrupt(io.EOF, exitCtrlD)
	menu.SetCommands(mainMenuCommands(app, agent))
	app.Start()
}

func exitCtrlD(c *console.Console) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Confirm exit (Y/y): ")
	text, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(text)

	if (answer == "Y") || (answer == "y") {
		os.Exit(0)
	}
}

func mainMenuCommands(app *console.Console, agent *llmagent.Agent) console.Commands {
	return func() *cobra.Command {
		rootCmd := &cobra.Command{
			Use: "ai-helper",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("XXXXX")
			},
		}

		versionCmd := &cobra.Command{
			Use:   "/version",
			Short: "Print the version number of Hugo",
			Long:  `All software has versions. This is Hugo's`,
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Printf("ai-helper %s\n", version.Version)
			},
		}
		msgCmd := &cobra.Command{
			Use: "msg",
			Run: func(cmd *cobra.Command, args []string) {
				fullCommand := fmt.Sprintf("%s", strings.Join(args, " "))

				// Add the command to the agent's message queue
				agent.AddMessage(chat.NewUserMessage(fullCommand))

				h := highlighter.NewHighlighter(os.Stdout)
				// Send the request to the AI agent
				_, cost, err := agent.Do(context.Background(), h)
				if err != nil {
					fmt.Printf("Error generating response: %v\n", err)
					return
				}

				fmt.Printf("Cost: %f\n", cost)
				// // Print AI agent responses
				// for _, resp := range responses {
				// 	fmt.Println(resp.Choice[0].Content[0].String())
				// }
			},
		}
		startMCP := &cobra.Command{
			Use: "/startmcp",
			Run: func(cmd *cobra.Command, args []string) {
				agent.StartMCP(context.Background())
			},
		}
		stopMCP := &cobra.Command{
			Use: "/stopmcp",
			Run: func(cmd *cobra.Command, args []string) {
				agent.StopMCP()
			},
		}
		// sessionCmd := &cobra.Command{
		// 	Use: "/session",
		// 	Run: func(cmd *cobra.Command, args []string) {
		// 		sessions, err := ai.ListAgents()
		// 		if err != nil {
		// 			fmt.Printf("%w\n", err)
		// 		}
		// 		for _, v := range sessions {
		// 			fmt.Println(v)
		// 		}
		// 	},
		// }

		rootCmd.AddCommand(versionCmd)
		rootCmd.AddCommand(startMCP)
		rootCmd.AddCommand(stopMCP)
		// rootCmd.AddCommand(sessionCmd)
		rootCmd.AddCommand(msgCmd)

		return rootCmd
	}
}
