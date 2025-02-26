package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/derricw/cwl/fetch"
	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

var queryString string
var startTime int64
var endTime int64

func init() {
	queryCmd.PersistentFlags().StringVarP(&queryString, "query", "q", "", "Query string.")
	queryCmd.PersistentFlags().Int64VarP(&startTime, "start-time", "s", 0, "Start time. Unix timestamp.")
	queryCmd.PersistentFlags().Int64VarP(&endTime, "end-time", "e", time.Now().Unix(), "End time. Unix timestamp.")
	rootCmd.AddCommand(queryCmd)
}

// Convert CloudWatch Logs query results to JSON format
func queryResultsToJSON(results [][]types.ResultField) ([]byte, error) {
	var formattedResults []map[string]string
	for _, row := range results {
		rowData := make(map[string]string)
		for _, field := range row {
			if field.Field != nil && field.Value != nil {
				rowData[*field.Field] = *field.Value
			}
		}
		formattedResults = append(formattedResults, rowData)
	}
	return json.Marshal(formattedResults) // Pretty-print JSON
}

var queryCmd = &cobra.Command{
	Use:   "query [logGroup]",
	Short: "query a log group",
	Long:  `Initiate a log insights query, wait for results, and write results to stdout.`,
	Example: `
Query a specific log group:

    cwl query /aws/batch/job -q "fields @timestamp, @message | sort @timestamp desc | limit 5"

Query all log groups:

    cwl query -q "fields @timestamp, @message | sort @timestamp desc" | limit 1000

Pass in a specific time range:

    cwl query -q "fields @timestamp, @message" -s $(date -d "2 weeks ago" +%s) -e $(date -d "yesterday" +%s)
  `,
	Args: cobra.MatchAll(cobra.MaximumNArgs(1)),
	Run: func(cmd *cobra.Command, args []string) {

		client, err := fetch.CreateClient(awsProfile)
		if err != nil {
			log.Fatal(err)
		}

		// Start the query
		startQueryInput := &cloudwatchlogs.StartQueryInput{
			StartTime: &startTime,
			EndTime:   &endTime,
		}

		if len(args) == 0 {
			// only way currently to query all log groups is to use SOURCE query
			queryString = "SOURCE logGroups() | " + queryString
		} else {
			// otherwise query all log groups passed in
			startQueryInput.LogGroupNames = strings.Split(args[0], ",")
		}
		startQueryInput.QueryString = &queryString

		ctx := context.TODO()
		startQueryOutput, err := client.StartQuery(ctx, startQueryInput)
		if err != nil {
			log.Fatal("Failed to start query:", err)
		}

		queryID := startQueryOutput.QueryId
		log.Println("Query started, ID:", *queryID)

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

			if queryResults.Status == types.QueryStatusComplete {
				break
			}

			log.Println("Waiting for query to complete... Status:", queryResults.Status)
		}

		jsonResults, err := queryResultsToJSON(queryResults.Results)
		if err != nil {
			log.Fatal("Failed to marshal query results: ", err)
		}
		fmt.Printf("%s\n", jsonResults)
	},
}
