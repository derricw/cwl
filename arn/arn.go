package arn

import (
	"fmt"
	"strings"
)

// StreamIdentifier represents a CloudWatch log stream
type StreamIdentifier struct {
	GroupName  string
	StreamName string
}

// ParseStreamArn extracts log group and stream names from an ARN
func ParseStreamArn(streamArn string) StreamIdentifier {
	streamArnTokens := strings.Split(streamArn, ":log-group:")
	streamNameTokens := strings.Split(streamArnTokens[1], ":log-stream:")
	return StreamIdentifier{
		GroupName:  streamNameTokens[0],
		StreamName: streamNameTokens[1],
	}
}

// CreateVirtualArn creates a virtual ARN from group and stream names
func CreateVirtualArn(groupName, streamName string) string {
	return fmt.Sprintf("_:log-group:%s:log-stream:%s", groupName, streamName)
}
