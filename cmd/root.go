package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/derricw/cwl/model"
	"github.com/derricw/cwl/provider/cloudwatch"
)

var jsonOutput bool
var awsProfile string
var logGroup string
var streamFilter string

func init() {
	rootCmd.PersistentFlags().StringVarP(&awsProfile, "profile", "p", "", "AWS Profile to use")
	rootCmd.Flags().StringVarP(&logGroup, "group", "g", "", "Log group to open directly into streams view")
	rootCmd.Flags().StringVarP(&streamFilter, "stream-filter", "s", "", "Filter streams by name (requires -g)")
}

var rootCmd = &cobra.Command{
	Use:   "cwl [subcommand]",
	Short: "Launch cwl tui",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		backend, err := cloudwatch.New(awsProfile)
		if err != nil {
			fmt.Printf("Error creating backend: %v", err)
			os.Exit(1)
		}
		deps := &model.Dependencies{Profile: awsProfile, Backend: backend}
		m := model.InitialModel(deps, logGroup, streamFilter)
		if _, ok := os.LookupEnv("DEBUG"); ok {
			logFile, err := os.OpenFile("messages.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
			if err != nil {
				os.Exit(1)
			}
			m.Log = logFile
		}
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
