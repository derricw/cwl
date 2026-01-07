package arn

import (
	"testing"
)

func TestParseStreamArn(t *testing.T) {
	tests := []struct {
		name           string
		arn            string
		expectedGroup  string
		expectedStream string
	}{
		{
			name:           "standard ARN",
			arn:            "arn:aws:logs:us-west-2:123456789012:log-group:/my/log/group:log-stream:my-stream",
			expectedGroup:  "/my/log/group",
			expectedStream: "my-stream",
		},
		{
			name:           "simple names",
			arn:            "arn:aws:logs:us-east-1:123:log-group:simple-group:log-stream:simple-stream",
			expectedGroup:  "simple-group",
			expectedStream: "simple-stream",
		},
		{
			name:           "nested group path",
			arn:            "arn:aws:logs:eu-west-1:456:log-group:/aws/lambda/my-function:log-stream:2023/12/01/[$LATEST]abcd1234",
			expectedGroup:  "/aws/lambda/my-function",
			expectedStream: "2023/12/01/[$LATEST]abcd1234",
		},
		{
			name:           "virtual ARN format",
			arn:            "_:log-group:/test/group:log-stream:test-stream",
			expectedGroup:  "/test/group",
			expectedStream: "test-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseStreamArn(tt.arn)
			
			if result.GroupName != tt.expectedGroup {
				t.Errorf("Expected group %q, got %q", tt.expectedGroup, result.GroupName)
			}
			
			if result.StreamName != tt.expectedStream {
				t.Errorf("Expected stream %q, got %q", tt.expectedStream, result.StreamName)
			}
		})
	}
}

func TestCreateVirtualArn(t *testing.T) {
	tests := []struct {
		name        string
		groupName   string
		streamName  string
		expectedArn string
	}{
		{
			name:        "simple names",
			groupName:   "test-group",
			streamName:  "test-stream",
			expectedArn: "_:log-group:test-group:log-stream:test-stream",
		},
		{
			name:        "nested group path",
			groupName:   "/aws/lambda/my-function",
			streamName:  "2023/12/01/stream",
			expectedArn: "_:log-group:/aws/lambda/my-function:log-stream:2023/12/01/stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CreateVirtualArn(tt.groupName, tt.streamName)
			
			if result != tt.expectedArn {
				t.Errorf("Expected ARN %q, got %q", tt.expectedArn, result)
			}
		})
	}
}
