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
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func init() {
	rootCmd.AddCommand(streamsCmd)
}

var streamsCmd = &cobra.Command{
	Use:   "streams [group]",
	Short: "list streams for a log group",
	Long:  `Lists all available streams for a log group.`,
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
			groupName := scanner.Text()
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
					fmt.Printf("%s::%s\n", groupName, *stream.LogStreamName)
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
