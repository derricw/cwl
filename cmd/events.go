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

var (
	follow bool
)

func init() {
	eventsCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "", false, "Output full json")
	eventsCmd.PersistentFlags().BoolVarP(&follow, "follow", "f", false, "Follow log stream")
	rootCmd.AddCommand(eventsCmd)
}

func writeEvent(event types.OutputLogEvent) {
	if jsonOutput {
		jsonData, err := json.Marshal(event)
		if err != nil {
			fmt.Println("Error marshaling to JSON:", err)
			return
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("%s\n", *event.Message)
	}
}

// extract log group and log stream name from a log stream ARN
func streamArnToName(streamArn string) (string, string) {
	streamArnTokens := strings.Split(streamArn, ":log-group:")
	streamNameTokens := strings.Split(streamArnTokens[1], ":log-stream:")
	return streamNameTokens[0], streamNameTokens[1]
}

var eventsCmd = &cobra.Command{
	Use:   "events [stream arn]",
	Short: "list events for log stream(s)",
	Long: `Lists events for a log stream. Stream arn can be passed as 
an argument or read from stdin.`,
	Args: cobra.MatchAll(cobra.MaximumNArgs(1)),
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

		scanner := bufio.NewScanner(readFrom)
		for scanner.Scan() {
			var nextToken *string
			streamArn := scanner.Text()
			groupName, streamName := streamArnToName(streamArn)
			for {

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				output, err := client.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
					LogGroupName:  &groupName,
					LogStreamName: &streamName,
					StartFromHead: aws.Bool(!follow), // in follow mode, we want the latest events
					NextToken:     nextToken,
					Limit:         aws.Int32(10000), // 10,000 is max allowed by AWS
				})
				if err != nil {
					log.Fatal(err)
				}
				cancel()
				for _, event := range output.Events {
					writeEvent(event)
				}

				if nextToken != nil && *nextToken == *output.NextForwardToken {
					if follow {
						time.Sleep(2 * time.Second)
					} else {
						break
					}
				}

				nextToken = output.NextForwardToken
			}
		}
	},
}
