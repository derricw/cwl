// Package cloudwatch implements provider.Backend for AWS CloudWatch Logs.
package cloudwatch

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/derricw/cwl/fetch"
	"github.com/derricw/cwl/interfaces"
	"github.com/derricw/cwl/provider"
)

type Backend struct {
	Client interfaces.CloudWatchLogsClient
}

func New(profile string) (*Backend, error) {
	client, err := fetch.CreateClient(profile)
	if err != nil {
		return nil, err
	}
	return &Backend{Client: client}, nil
}

func (b *Backend) FetchGroups(pattern string) ([]provider.LogGroup, error) {
	groups, err := fetch.FetchLogGroups(b.Client, pattern)
	if err != nil {
		return nil, err
	}
	result := make([]provider.LogGroup, len(groups))
	for i, g := range groups {
		result[i] = provider.LogGroup{
			Name: *g.LogGroupName,
			Desc: *g.LogGroupArn,
		}
	}
	return result, nil
}

func (b *Backend) FetchStreamsStreaming(group string, callback func([]provider.LogStream) error) error {
	return fetch.FetchLogStreamsStreaming(b.Client, group, func(streams []types.LogStream) error {
		converted := make([]provider.LogStream, len(streams))
		for i, s := range streams {
			converted[i] = provider.LogStream{Name: *s.LogStreamName}
			if s.LastEventTimestamp != nil {
				t := time.Unix(0, *s.LastEventTimestamp*int64(time.Millisecond))
				converted[i].LastEventTime = &t
			}
		}
		return callback(converted)
	})
}

func (b *Backend) FetchEventsStreaming(group, stream string, callback func([]provider.LogEvent) error) error {
	return fetch.FetchLogEventsStreaming(b.Client, group, stream, func(events []types.OutputLogEvent) error {
		return callback(convertEvents(events))
	})
}

func (b *Backend) FetchLastEvents(group, stream string, limit int) ([]provider.LogEvent, error) {
	events, err := fetch.FetchLastLogEvents(b.Client, group, stream, int32(limit))
	if err != nil {
		return nil, err
	}
	return convertEvents(events), nil
}

func (b *Backend) FetchNewEvents(group, stream string, since *time.Time) ([]provider.LogEvent, error) {
	if since == nil {
		return nil, nil
	}
	sinceMs := since.UnixMilli()
	events, err := fetch.FetchNewLogEvents(b.Client, group, stream, &sinceMs)
	if err != nil {
		return nil, err
	}
	return convertEvents(events), nil
}

func convertEvents(events []types.OutputLogEvent) []provider.LogEvent {
	result := make([]provider.LogEvent, len(events))
	for i, e := range events {
		if e.Message != nil {
			result[i].Message = *e.Message
		}
		if e.Timestamp != nil {
			t := time.Unix(0, *e.Timestamp*int64(time.Millisecond))
			result[i].Timestamp = &t
		}
	}
	return result
}
