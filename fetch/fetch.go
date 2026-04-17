package fetch

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/derricw/cwl/interfaces"
)

// CreateClient builds a CloudWatch Logs client with short HTTP timeouts.
// The short dial timeout (2s) ensures that credential resolution fails fast
// on non-EC2 machines where IMDS (169.254.169.254) is unreachable, rather than
// hanging for minutes on retries. The SDK's default retry behavior for
// throttling is preserved since we don't override MaxAttempts.
func CreateClient(profileName string) (interfaces.CloudWatchLogsClient, error) {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 2 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 2 * time.Second,
		},
	}
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(profileName),
		config.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, err
	}
	return cloudwatchlogs.NewFromConfig(cfg), nil
}

func FetchLogGroups(client interfaces.CloudWatchLogsClient) ([]types.LogGroup, error) {
	groups := make([]types.LogGroup, 0)
	var nextToken *string

	for {
		output, err := client.DescribeLogGroups(context.TODO(), &cloudwatchlogs.DescribeLogGroupsInput{NextToken: nextToken})
		if err != nil {
			return nil, err
		}
		groups = append(groups, output.LogGroups...)
		if output.NextToken != nil {
			nextToken = output.NextToken
		} else {
			break
		}
	}
	return groups, nil
}

func FetchLogStreams(client interfaces.CloudWatchLogsClient, logGroupName string, maxResults int) ([]types.LogStream, error) {
	streams := make([]types.LogStream, 0)
	var nextToken *string

	for {
		output, err := client.DescribeLogStreams(context.TODO(), &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: &logGroupName,
			Limit:        aws.Int32(50),
			OrderBy:      types.OrderByLastEventTime,
			Descending:   aws.Bool(true),
			NextToken:    nextToken,
		})
		if err != nil {
			return nil, err
		}
		streams = append(streams, output.LogStreams...)
		if len(streams) >= maxResults {
			break
		}
		if output.NextToken != nil {
			nextToken = output.NextToken
		} else {
			break
		}
	}
	return streams, nil
}

var ErrMaxStreamsReached = fmt.Errorf("max streams reached")

func FetchLogStreamsStreaming(client interfaces.CloudWatchLogsClient, logGroupName string, callback func([]types.LogStream) error) error {
	var nextToken *string

	for {
		output, err := client.DescribeLogStreams(context.TODO(), &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: &logGroupName,
			Limit:        aws.Int32(50),
			OrderBy:      types.OrderByLastEventTime,
			Descending:   aws.Bool(true),
			NextToken:    nextToken,
		})
		if err != nil {
			return err
		}
		if len(output.LogStreams) > 0 {
			if err := callback(output.LogStreams); err != nil {
				return err
			}
		}
		if output.NextToken != nil {
			nextToken = output.NextToken
		} else {
			break
		}
	}
	return nil
}

func FetchLastLogEvents(client interfaces.CloudWatchLogsClient, logGroupName, logStreamName string, limit int32) ([]types.OutputLogEvent, error) {
	output, err := client.GetLogEvents(context.TODO(), &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  &logGroupName,
		LogStreamName: &logStreamName,
		StartFromHead: aws.Bool(false),
		Limit:         aws.Int32(limit),
	})
	if err != nil {
		return nil, err
	}
	return output.Events, nil
}

func FetchLogEvents(client interfaces.CloudWatchLogsClient, logGroupName, logStreamName string) ([]types.OutputLogEvent, error) {
	events := make([]types.OutputLogEvent, 0)
	var nextToken *string

	for {
		output, err := client.GetLogEvents(context.TODO(), &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  &logGroupName,
			LogStreamName: &logStreamName,
			StartFromHead: aws.Bool(true),
			//Limit # of events?
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		events = append(events, output.Events...)
		if output.NextForwardToken == nil {
			break
		} else if nextToken == nil || *output.NextForwardToken == *nextToken {
			nextToken = output.NextForwardToken
		} else {
			break
		}
	}
	return events, nil
}

func FetchLogEventsStreaming(client interfaces.CloudWatchLogsClient, logGroupName, logStreamName string, callback func([]types.OutputLogEvent) error) error {
	var nextToken *string

	for {
		output, err := client.GetLogEvents(context.TODO(), &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  &logGroupName,
			LogStreamName: &logStreamName,
			StartFromHead: aws.Bool(true),
			Limit:         aws.Int32(10000),
			NextToken:     nextToken,
		})
		if err != nil {
			return err
		}
		if len(output.Events) > 0 {
			if err := callback(output.Events); err != nil {
				return err
			}
		}
		if output.NextForwardToken == nil {
			break
		} else if nextToken == nil || *output.NextForwardToken == *nextToken {
			nextToken = output.NextForwardToken
		} else {
			break
		}
	}
	return nil
}

func FetchNewLogEvents(client interfaces.CloudWatchLogsClient, logGroupName, logStreamName string, startTime *int64) ([]types.OutputLogEvent, error) {
	if startTime == nil {
		return nil, nil
	}
	output, err := client.GetLogEvents(context.TODO(), &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  &logGroupName,
		LogStreamName: &logStreamName,
		StartTime:     aws.Int64(*startTime + 1),
		StartFromHead: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	return output.Events, nil
}
