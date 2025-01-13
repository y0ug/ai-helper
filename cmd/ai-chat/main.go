package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/reeflective/console"
)

func main() {
}

func StartConsole() {
	app := console.New("example")

	store := make(map[string]string)

	app.NewlineBefore = true
	app.NewlineAfter = true
	menu := app.ActiveMenu()
	menu.AddInterrupt(io.EOF, exitCtrlD)
	menu.SetCommands(mainMenuCommands(app, store))
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
