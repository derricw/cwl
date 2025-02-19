package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/derricw/cwl/fetch"
	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

func init() {
	rootCmd.AddCommand(eventsCmd)
}

var eventsCmd = &cobra.Command{
	Use:   "events [stream]",
	Short: "list events for a log stream",
	Long:  `Lists events for a log stream.`,
	Args:  cobra.MatchAll(cobra.MaximumNArgs(1)),
	Run: func(cmd *cobra.Command, args []string) {

		client, err := fetch.CreateClient()
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
			streamText := scanner.Text()
			streamTokens := strings.Split(streamText, "::")
			groupName, streamName := streamTokens[0], streamTokens[1]
			for {
				output, err := client.GetLogEvents(context.TODO(), &cloudwatchlogs.GetLogEventsInput{
					LogGroupName:  &groupName,
					LogStreamName: &streamName,
					StartFromHead: aws.Bool(true),
					NextToken:     nextToken,
				})
				if err != nil {
					log.Fatal(err)
				}
				for _, event := range output.Events {
					fmt.Printf("%s\n", *event.Message)
				}
				if output.NextForwardToken == nil {
					break
				} else if output.NextForwardToken != output.NextForwardToken {
					nextToken = output.NextForwardToken
				} else {
					break
				}
			}
		}
	},
}
