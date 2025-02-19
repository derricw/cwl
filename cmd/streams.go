package cmd

import (
	"fmt"
	"log"

	"github.com/derricw/cwl/fetch"
	"github.com/spf13/cobra"

	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func init() {
	rootCmd.AddCommand(streamsCmd)
}

var streamsCmd = &cobra.Command{
	Use:   "streams [group]",
	Short: "list streams for a log group",
	Long:  `Lists all available streams for a log group`,
	Args:  cobra.MatchAll(cobra.ExactArgs(1)),
	Run: func(cmd *cobra.Command, args []string) {

		groupName := args[0]

		client, err := fetch.CreateClient()
		if err != nil {
			log.Fatal(err)
		}

		/*
			groups, err := fetch.FetchLogStreams(client, groupName, 100000000)
			if err != nil {
				log.Fatal(err)
			}
			for _, group := range groups {
				fmt.Printf("%s\n", *group.LogStreamName)
			}
		*/

		var nextToken *string

		for {
			output, err := client.DescribeLogStreams(context.TODO(), &cloudwatchlogs.DescribeLogStreamsInput{
				LogGroupName: &groupName,
				Limit:        aws.Int32(50),
				OrderBy:      types.OrderByLastEventTime,
				Descending:   aws.Bool(true),
				NextToken:    nextToken,
			})
			if err != nil {
				log.Fatal(err)
			}
			for _, stream := range output.LogStreams {
				fmt.Printf("%s\n", *stream.LogStreamName)
			}
			if output.NextToken != nil {
				nextToken = output.NextToken
			} else {
				break
			}
		}

	},
}
