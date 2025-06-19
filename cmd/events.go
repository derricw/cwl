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
	"sync"
	"time"

	"github.com/derricw/cwl/fetch"
	"github.com/spf13/cobra"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

var (
	follow             bool
	minPollingInterval = 1 * time.Second  // Minimum interval for polling events
	maxPollingInterval = 16 * time.Second // Maximum interval for polling events
)

func init() {
	eventsCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "", false, "Output full json")
	eventsCmd.PersistentFlags().BoolVarP(&follow, "follow", "f", false, "Follow log stream")
	rootCmd.AddCommand(eventsCmd)
}

// extract log group and log stream name from a log stream ARN
func streamArnToName(streamArn string) (string, string) {
	streamArnTokens := strings.Split(streamArn, ":log-group:")
	streamNameTokens := strings.Split(streamArnTokens[1], ":log-stream:")
	return streamNameTokens[0], streamNameTokens[1]
}

// read events from a channel and print them to stdout
func writeEvents(events <-chan types.OutputLogEvent) {
	for event := range events {
		if jsonOutput {
			jsonData, err := json.Marshal(event)
			if err != nil {
				fmt.Println("Error marshaling to JSON:", err)
				continue
			}
			fmt.Println(string(jsonData))
		} else {
			fmt.Printf("%s\n", *event.Message)
		}
	}
}

// requestEvents fetches events from a log stream and sends them to the output channel.
func requestEvents(client *cloudwatchlogs.Client, groupName, streamName string, outputChan chan types.OutputLogEvent) error {
	var nextToken *string
	var interval time.Duration
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		input := &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  &groupName,
			LogStreamName: &streamName,
			StartFromHead: aws.Bool(!follow), // in follow mode, we want the latest events
			NextToken:     nextToken,
			Limit:         aws.Int32(10000), // 10,000 is max allowed by AWS
		}
		output, err := client.GetLogEvents(ctx, input)
		if err != nil {
			cancel()
			return err
		}
		cancel()
		for _, event := range output.Events {
			outputChan <- event
		}

		if len(output.Events) > 0 {
			interval = minPollingInterval
		} else {
			interval = min(maxPollingInterval, interval*2)
		}

		if nextToken != nil && *nextToken == *output.NextForwardToken {
			if follow {
				time.Sleep(interval)
			} else {
				break
			}
		}
		nextToken = output.NextForwardToken
	}
	return nil
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

		eventChannel := make(chan types.OutputLogEvent, 10000)
		var processWg sync.WaitGroup
		processWg.Add(1)
		go func() {
			defer processWg.Done()
			writeEvents(eventChannel)
		}()

		var wg sync.WaitGroup
		scanner := bufio.NewScanner(readFrom)
		for scanner.Scan() {
			streamArn := scanner.Text()
			groupName, streamName := streamArnToName(streamArn)
			wg.Add(1)
			go func(g, s string, ech chan types.OutputLogEvent) {
				defer wg.Done()
				requestEvents(client, g, s, ech)
			}(groupName, streamName, eventChannel)
		}

		// wait on everything to finish
		wg.Wait()
		close(eventChannel)
		processWg.Wait()
	},
}
