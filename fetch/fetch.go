package fetch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

func CreateClient() (*cloudwatchlogs.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	return cloudwatchlogs.NewFromConfig(cfg), nil
}

func FetchLogGroups(client *cloudwatchlogs.Client) ([]types.LogGroup, error) {
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

func FetchLogStreams(client *cloudwatchlogs.Client, logGroupName string) ([]types.LogStream, error) {
	streams := make([]types.LogStream, 0)
	var nextToken *string

	maxLogSteams := 300

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
		if len(streams) >= maxLogSteams {
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

func FetchLogEvents(client *cloudwatchlogs.Client, logGroupName, logStreamName string) ([]types.OutputLogEvent, error) {
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
		} else if output.NextForwardToken != output.NextForwardToken {
			nextToken = output.NextForwardToken
		} else {
			break
		}
	}
	return events, nil
}
