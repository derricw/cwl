package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/derricw/cwl/fetch"
	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

var groupFilter string

func init() {
	groupsCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "", false, "Output full json")
	groupsCmd.PersistentFlags().StringVarP(&groupFilter, "filter", "f", "", "Server-side pattern to filter log groups")
	rootCmd.AddCommand(groupsCmd)
}

func writeGroup(group types.LogGroup) {
	if jsonOutput {
		jsonData, err := json.Marshal(group)
		if err != nil {
			fmt.Println("Error marshaling to JSON:", err)
			return
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("%s\n", *group.LogGroupName)
	}
}

var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "list groups",
	Long:  `Lists all available log groups`,
	Run: func(cmd *cobra.Command, args []string) {

		client, err := fetch.CreateClient(awsProfile)
		if err != nil {
			log.Fatal(err)
		}

		var nextToken *string

		for {

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			input := &cloudwatchlogs.DescribeLogGroupsInput{NextToken: nextToken}
			if groupFilter != "" {
				input.LogGroupNamePattern = &groupFilter
			}
			output, err := client.DescribeLogGroups(ctx, input)
			if err != nil {
				log.Fatal(err)
			}
			cancel()
			for _, group := range output.LogGroups {
				writeGroup(group)
			}
			if output.NextToken != nil {
				nextToken = output.NextToken
			} else {
				break
			}
		}
	},
}
