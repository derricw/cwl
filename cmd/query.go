package cmd

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/derricw/cwl/fetch"
	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func init() {
	rootCmd.AddCommand(queryCmd)
}

var queryCmd = &cobra.Command{
	Use:   "query [logGroup]",
	Short: "query a log group",
	Long:  ``,
	Args:  cobra.MatchAll(cobra.ExactArgs(1)),
	Run: func(cmd *cobra.Command, args []string) {

		client, err := fetch.CreateClient(awsProfile)
		if err != nil {
			log.Fatal(err)
		}
		// Define query parameters
		logGroupName := args[0]
		queryString := `fields @timestamp, @message | sort @timestamp desc | limit 5`
		//startTime := time.Now().Add(-time.Hour * 1).Unix() // 1 hour ago
		var startTime int64 = 0
		endTime := time.Now().Unix() // Now

		// Start the query
		startQueryInput := &cloudwatchlogs.StartQueryInput{
			LogGroupName: &logGroupName,
			QueryString:  &queryString,
			StartTime:    &startTime,
			EndTime:      &endTime,
		}
		ctx := context.TODO()
		startQueryOutput, err := client.StartQuery(ctx, startQueryInput)
		if err != nil {
			log.Fatal("Failed to start query:", err)
		}

		queryID := startQueryOutput.QueryId
		fmt.Println("Query started, ID:", *queryID)

		// Poll for query results
		var queryResults *cloudwatchlogs.GetQueryResultsOutput
		for {
			time.Sleep(2 * time.Second) // Wait before polling

			queryResults, err = client.GetQueryResults(ctx, &cloudwatchlogs.GetQueryResultsInput{
				QueryId: queryID,
			})
			if err != nil {
				log.Fatal("Failed to get query results:", err)
			}

			// Check if query is complete
			if queryResults.Status == types.QueryStatusComplete {
				break
			}

			fmt.Println("Waiting for query to complete... Status:", queryResults.Status)
		}

		// Print query results
		fmt.Println("Query results:")
		for _, row := range queryResults.Results {
			for _, field := range row {
				fmt.Printf("%s: %s | ", *field.Field, *field.Value)
			}
			fmt.Println()
		}
	},
}
