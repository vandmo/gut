package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vandmo/gut/cmd"
	"log"
	"os"
)

func main() {
	logfilePath := os.Getenv("GUT_LOG")
	if logfilePath != "" {
		if _, err := tea.LogToFile(logfilePath, "debug"); err != nil {
			log.Fatal(err)
		}
	}
	cmd.Execute()
}
