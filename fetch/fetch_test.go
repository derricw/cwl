package fetch

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/derricw/cwl/interfaces"
)

// MockCloudWatchLogsClient implements the CloudWatchLogsClient interface for testing
type MockCloudWatchLogsClient struct {
	LogGroups []types.LogGroup
	Error     error
}

func (m *MockCloudWatchLogsClient) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return &cloudwatchlogs.DescribeLogGroupsOutput{
		LogGroups: m.LogGroups,
	}, nil
}

func (m *MockCloudWatchLogsClient) DescribeLogStreams(ctx context.Context, params *cloudwatchlogs.DescribeLogStreamsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	return nil, nil
}

func (m *MockCloudWatchLogsClient) GetLogEvents(ctx context.Context, params *cloudwatchlogs.GetLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetLogEventsOutput, error) {
	return nil, nil
}

func (m *MockCloudWatchLogsClient) CreateLogStream(ctx context.Context, params *cloudwatchlogs.CreateLogStreamInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogStreamOutput, error) {
	return nil, nil
}

func (m *MockCloudWatchLogsClient) PutLogEvents(ctx context.Context, params *cloudwatchlogs.PutLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutLogEventsOutput, error) {
	return nil, nil
}

func (m *MockCloudWatchLogsClient) StartQuery(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error) {
	return nil, nil
}

func (m *MockCloudWatchLogsClient) GetQueryResults(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error) {
	return nil, nil
}

// Ensure MockCloudWatchLogsClient implements the interface
var _ interfaces.CloudWatchLogsClient = (*MockCloudWatchLogsClient)(nil)

func TestFetchLogGroups(t *testing.T) {
	// Test successful fetch
	mockClient := &MockCloudWatchLogsClient{
		LogGroups: []types.LogGroup{
			{LogGroupName: stringPtr("/test/group1")},
			{LogGroupName: stringPtr("/test/group2")},
		},
	}

	groups, err := FetchLogGroups(mockClient)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(groups) != 2 {
		t.Fatalf("Expected 2 groups, got %d", len(groups))
	}

	if *groups[0].LogGroupName != "/test/group1" {
		t.Errorf("Expected first group name '/test/group1', got %s", *groups[0].LogGroupName)
	}
}

func stringPtr(s string) *string {
	return &s
}
