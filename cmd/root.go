package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/derricw/cwl/model"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "cwl",
	Short: "Launch cwl tui",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		var log *os.File
		if _, ok := os.LookupEnv("DEBUG"); ok {
			var err error
			log, err = os.OpenFile("messages.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				os.Exit(1)
			}
		}
		m := model.InitialModel()
		m.Log = log
		p := tea.NewProgram(m, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
