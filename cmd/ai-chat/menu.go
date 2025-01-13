package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/reeflective/console"
)

// var store map[string]string
//
// func init() {
// 	store = make(map[string]string)
// }

func mainMenuCommands(app *console.Console, store map[string]string) console.Commands {
	return func() *cobra.Command {
		rootCmd := &cobra.Command{}

		versionCmd := &cobra.Command{
			Use:   "version",
			Short: "Print the version number of Hugo",
			Long:  `All software has versions. This is Hugo's`,
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Hugo Static Site Generator v0.9 -- HEAD")
			},
		}

		setCmd := &cobra.Command{
			Use:   "set",
			Short: "Set a variable name",
			Args:  cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Setting variable", args[0], "to", args[1])
				store[args[0]] = args[1]
			},
		}

		printCmd := &cobra.Command{
			Use:  "print",
			Args: cobra.ExactArgs(1),
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Printing variable", args[0], "to", store[args[0]])
				fmt.Println(store)
			},
		}

		rootCmd.AddCommand(versionCmd)
		rootCmd.AddCommand(setCmd)
		rootCmd.AddCommand(printCmd)

		return rootCmd
	}
}
