package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/derricw/cwl/fetch"
	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

var prefix string

func init() {
	streamsCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "", false, "Output full json")
	streamsCmd.PersistentFlags().BoolVarP(&follow, "follow", "f", false, "Keep checking for new streams with events")
	streamsCmd.PersistentFlags().StringVar(&prefix, "prefix", "", "Filter streams by prefix")
	rootCmd.AddCommand(streamsCmd)
}

func writeStream(stream types.LogStream) {
	if jsonOutput {
		jsonData, err := json.Marshal(stream)
		if err != nil {
			fmt.Println("Error marshaling to JSON:", err)
			return
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("%s\n", *stream.Arn)
	}
}

var streamsCmd = &cobra.Command{
	Use:   "streams [group]",
	Short: "List stream arns for a log group",
	Long:  `Lists all available streams for a log group.`,
	Args:  cobra.MatchAll(cobra.MaximumNArgs(1)),
	Run: func(cmd *cobra.Command, args []string) {

		client, err := fetch.CreateClient(awsProfile)
		if err != nil {
			log.Fatal(err)
		}

		var readFrom io.Reader

		if len(args) == 0 {
			// read groups from stdin
			readFrom = os.Stdin
		} else {
			// a group was passed on command line
			readFrom = strings.NewReader(args[0])
		}

		seen := map[string]struct{}{}

		scanner := bufio.NewScanner(readFrom)
		for scanner.Scan() {
			groupName := scanner.Text()
			var nextToken *string
			start := time.Now().UnixNano() / 1000000
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				input := &cloudwatchlogs.DescribeLogStreamsInput{
					LogGroupName: &groupName,
					Limit:        aws.Int32(50),
					OrderBy:      types.OrderByLastEventTime,
					Descending:   aws.Bool(true),
					NextToken:    nextToken,
				}
				if prefix != "" {
					input.LogStreamNamePrefix = &prefix
					input.OrderBy = types.OrderByLogStreamName
				}
				output, err := client.DescribeLogStreams(ctx, input)
				if err != nil {
					log.Fatal(err)
				}
				cancel()
				for _, stream := range output.LogStreams {
					if !follow || stream.LastIngestionTime == nil || *stream.LastIngestionTime > start {
						if _, found := seen[*stream.LogStreamName]; !found {
							writeStream(stream)
							seen[*stream.LogStreamName] = struct{}{}
						}
					}
				}
				if output.NextToken != nil {
					nextToken = output.NextToken
				} else {
					if follow {
						time.Sleep(time.Second * 10)
					} else {
						break
					}
				}
			}
		}
	},
}
