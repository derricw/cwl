package cmd

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// MockQueryClient implements the CloudWatchLogsClient interface for query testing
type MockQueryClient struct {
	QueryID     string
	QueryStatus types.QueryStatus
	Results     [][]types.ResultField
	Error       error
}

func (m *MockQueryClient) StartQuery(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return &cloudwatchlogs.StartQueryOutput{
		QueryId: &m.QueryID,
	}, nil
}

func (m *MockQueryClient) GetQueryResults(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return &cloudwatchlogs.GetQueryResultsOutput{
		Status:  m.QueryStatus,
		Results: m.Results,
	}, nil
}

// Stub implementations for other interface methods
func (m *MockQueryClient) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	return nil, nil
}

func (m *MockQueryClient) DescribeLogStreams(ctx context.Context, params *cloudwatchlogs.DescribeLogStreamsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	return nil, nil
}

func (m *MockQueryClient) GetLogEvents(ctx context.Context, params *cloudwatchlogs.GetLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetLogEventsOutput, error) {
	return nil, nil
}

func (m *MockQueryClient) CreateLogStream(ctx context.Context, params *cloudwatchlogs.CreateLogStreamInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogStreamOutput, error) {
	return nil, nil
}

func (m *MockQueryClient) PutLogEvents(ctx context.Context, params *cloudwatchlogs.PutLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutLogEventsOutput, error) {
	return nil, nil
}

func TestQueryResultsToJSON(t *testing.T) {
	timestamp, message := "@timestamp", "@message"
	timestampVal, messageVal := "2023-01-01T10:00:00Z", "Test message"
	level, levelInfo, levelError := "level", "INFO", "ERROR"
	msg, msg1, msg2 := "message", "First log", "Second log"

	tests := []struct {
		name     string
		results  [][]types.ResultField
		expected string
	}{
		{
			name: "simple results",
			results: [][]types.ResultField{
				{
					{Field: &timestamp, Value: &timestampVal},
					{Field: &message, Value: &messageVal},
				},
			},
			expected: `[{"@message":"Test message","@timestamp":"2023-01-01T10:00:00Z"}]`,
		},
		{
			name:     "empty results",
			results:  [][]types.ResultField{},
			expected: `null`,
		},
		{
			name: "multiple rows",
			results: [][]types.ResultField{
				{
					{Field: &level, Value: &levelInfo},
					{Field: &msg, Value: &msg1},
				},
				{
					{Field: &level, Value: &levelError},
					{Field: &msg, Value: &msg2},
				},
			},
			expected: `[{"level":"INFO","message":"First log"},{"level":"ERROR","message":"Second log"}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := queryResultsToJSON(tt.results)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(result))
			}
		})
	}
}
