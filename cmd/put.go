package cmd

import (
	"bufio"
	"context"
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
	rootCmd.AddCommand(putCmd)
}

func ensureLogStreamExists(client *cloudwatchlogs.Client, logGroupName, logStreamName string) error {
	// Try to describe the log stream to check if it exists
	output, err := client.DescribeLogStreams(context.TODO(), &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName:        aws.String(logGroupName),
		LogStreamNamePrefix: aws.String(logStreamName),
	})
	if len(output.LogStreams) == 0 {
		log.Printf("Log stream did not exist: %s Creating...", logStreamName)
		_, err = client.CreateLogStream(context.TODO(), &cloudwatchlogs.CreateLogStreamInput{
			LogGroupName:  aws.String(logGroupName),
			LogStreamName: aws.String(logStreamName),
		})
		return err
	}
	return err
}

var putCmd = &cobra.Command{
	Use:   "put [stream arn] [event]",
	Short: "put events for log stream",
	Long:  `Put events for a log stream. Can stream events from stdin`,
	Args:  cobra.MatchAll(cobra.MinimumNArgs(1)),
	Run: func(cmd *cobra.Command, args []string) {

		client, err := fetch.CreateClient(awsProfile)
		if err != nil {
			log.Fatal(err)
		}

		var readFrom io.Reader
		var streamArn string

		if len(args) == 1 {
			// read events from stdin
			streamArn = args[0]
			readFrom = os.Stdin
		} else if len(args) == 2 {
			streamArn = args[0]
			readFrom = strings.NewReader(args[0])
		} else {
			log.Fatal("Expected no more than 2 args")
		}
		groupName, streamName := streamArnToName(streamArn)
		err = ensureLogStreamExists(client, groupName, streamName)
		if err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(readFrom)
		for scanner.Scan() {
			event := scanner.Text()
			input := &cloudwatchlogs.PutLogEventsInput{
				LogGroupName:  aws.String(groupName),
				LogStreamName: aws.String(streamName),
				LogEvents: []types.InputLogEvent{
					// TODO: batching is supported up to 10000 events or 1MB
					{
						Message:   aws.String(event),
						Timestamp: aws.Int64(time.Now().UnixNano() / 1000000), // Convert to milliseconds
					},
				},
			}
			_, err = client.PutLogEvents(context.TODO(), input)
			if err != nil {
				log.Fatal(err)
			}
		}
	},
}
