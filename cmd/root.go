package cmd

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/derricw/cwl/model"
	"github.com/derricw/cwl/provider"
	"github.com/derricw/cwl/provider/cloudwatch"
	"github.com/derricw/cwl/provider/mlflow"
)

var jsonOutput bool
var awsProfile string
var logGroup string
var streamFilter string
var mlflowURL string
var mlflowARN string

func init() {
	rootCmd.PersistentFlags().StringVarP(&awsProfile, "profile", "p", "", "AWS Profile to use")
	rootCmd.Flags().StringVarP(&logGroup, "group", "g", "", "Log group to open directly into streams view")
	rootCmd.Flags().StringVarP(&streamFilter, "stream-filter", "s", "", "Filter streams by name (requires -g)")
	rootCmd.Flags().StringVar(&mlflowURL, "mlflow-url", "", "MLflow tracking server URL (implies mlflow backend)")
	rootCmd.Flags().StringVar(&mlflowARN, "mlflow-arn", "", "SageMaker MLflow tracking server ARN (implies mlflow backend)")
}

// createBackend resolves the backend from flags and env vars.
// Priority: --mlflow-arn > --mlflow-url > MLFLOW_TRACKING_URI env > CloudWatch default.
func createBackend() (provider.Backend, error) {
	// Explicit flags take priority
	if mlflowARN != "" {
		return mlflow.NewFromSageMakerARN(mlflowARN, awsProfile)
	}
	if mlflowURL != "" {
		return mlflow.New(mlflowURL), nil
	}

	// Check MLFLOW_TRACKING_URI env var
	if uri := os.Getenv("MLFLOW_TRACKING_URI"); uri != "" {
		if strings.HasPrefix(uri, "arn:") {
			return mlflow.NewFromSageMakerARN(uri, awsProfile)
		}
		return mlflow.New(uri), nil
	}

	// Default to CloudWatch
	return cloudwatch.New(awsProfile)
}

var rootCmd = &cobra.Command{
	Use:   "cwl [subcommand]",
	Short: "Launch cwl tui",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		backend, err := createBackend()
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
