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

func init() {
	streamsCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "", false, "Output full json")
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

		var nextToken *string

		scanner := bufio.NewScanner(readFrom)
		for scanner.Scan() {
			groupName := scanner.Text()
			for {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				output, err := client.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
					LogGroupName: &groupName,
					Limit:        aws.Int32(50),
					OrderBy:      types.OrderByLastEventTime,
					Descending:   aws.Bool(true),
					NextToken:    nextToken,
				})
				if err != nil {
					log.Fatal(err)
				}
				cancel()
				for _, stream := range output.LogStreams {
					writeStream(stream)
				}
				if output.NextToken != nil {
					nextToken = output.NextToken
				} else {
					break
				}
			}
		}
	},
}
