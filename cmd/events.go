package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
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
	noColor            bool
)

func init() {
	eventsCmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "", false, "Output full json")
	eventsCmd.PersistentFlags().BoolVarP(&follow, "follow", "f", false, "Follow log stream")
	eventsCmd.PersistentFlags().BoolVarP(&noColor, "no-color", "c", false, "disable colored output")
	rootCmd.AddCommand(eventsCmd)
}

// Event is an event that we can write to stdout
// It contains a Cloudwatch event and a color
type Event struct {
	cwEvent types.OutputLogEvent
	color   *lipgloss.Color
}

// Render writes the event to stdout, taking into
// consideration the color and json output flag
func (e *Event) Render() {
	var buffer string
	if jsonOutput {
		jsonData, err := json.Marshal(e.cwEvent)
		if err != nil {
			fmt.Println("Error marshalling to JSON:", err)
		}
		buffer = string(jsonData)
	} else {
		buffer = *e.cwEvent.Message
	}

	if noColor || e.color == nil {
		fmt.Println(buffer)
	} else {
		style := lipgloss.NewStyle().Foreground(e.color)
		fmt.Println(style.Render(buffer))
	}
}

// extract log group and log stream name from a log stream ARN
func streamArnToName(streamArn string) (string, string) {
	streamArnTokens := strings.Split(streamArn, ":log-group:")
	streamNameTokens := strings.Split(streamArnTokens[1], ":log-stream:")
	return streamNameTokens[0], streamNameTokens[1]
}

// read events from a channel and render them as they come in
func writeEvents(events <-chan Event) {
	for event := range events {
		event.Render()
	}
}

// requestEvents fetches events from a log stream and sends them to the output channel.
func requestEvents(client *cloudwatchlogs.Client, groupName, streamName string, outputChan chan Event, color *lipgloss.Color) error {
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
			outputChan <- Event{event, color}
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
			// read streams from stdin
			readFrom = os.Stdin
		} else {
			// a stream was passed on command line
			readFrom = strings.NewReader(args[0])
		}

		eventChannel := make(chan Event, 10000)
		var processWg sync.WaitGroup
		processWg.Add(1)
		go func() {
			defer processWg.Done()
			writeEvents(eventChannel)
		}()

		var wg sync.WaitGroup
		scanner := bufio.NewScanner(readFrom)
		streamIdx := 0
		var color *lipgloss.Color
		if !noColor { // shuffle colors
			rand.Shuffle(len(colors), func(i, j int) { colors[i], colors[j] = colors[j], colors[i] })
		}

		// iterate over streams and request events for each
		for scanner.Scan() {
			streamArn := scanner.Text()
			groupName, streamName := streamArnToName(streamArn)
			if streamIdx != 0 {
				color = &colors[streamIdx%len(colors)]
			}
			wg.Add(1)
			go func(g, s string, ech chan Event, c *lipgloss.Color) {
				defer wg.Done()
				requestEvents(client, g, s, ech, c)
			}(groupName, streamName, eventChannel, color)
			streamIdx++
		}

		// wait on everything to finish
		wg.Wait()
		close(eventChannel)
		processWg.Wait()
	},
}
