package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

// entrypoint
func main() {
	var log *os.File
	if _, ok := os.LookupEnv("DEBUG"); ok {
		var err error
		log, err = os.OpenFile("messages.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			os.Exit(1)
		}
	}
	m := initialModel()
	m.log = log
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
