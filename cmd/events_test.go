package cmd

import (
	"testing"

	"github.com/derricw/cwl/arn"
)

func TestStreamArnToName(t *testing.T) {
	tests := []struct {
		name           string
		arnStr         string
		expectedGroup  string
		expectedStream string
	}{
		{
			name:           "standard ARN",
			arnStr:         "arn:aws:logs:us-west-2:123456789012:log-group:/my/log/group:log-stream:my-stream",
			expectedGroup:  "/my/log/group",
			expectedStream: "my-stream",
		},
		{
			name:           "simple names",
			arnStr:         "arn:aws:logs:us-east-1:123:log-group:simple-group:log-stream:simple-stream",
			expectedGroup:  "simple-group",
			expectedStream: "simple-stream",
		},
		{
			name:           "nested group path",
			arnStr:         "arn:aws:logs:eu-west-1:456:log-group:/aws/lambda/my-function:log-stream:2023/12/01/[$LATEST]abcd1234",
			expectedGroup:  "/aws/lambda/my-function",
			expectedStream: "2023/12/01/[$LATEST]abcd1234",
		},
		{
			name:           "virtual ARN format",
			arnStr:         "_:log-group:/test/group:log-stream:test-stream",
			expectedGroup:  "/test/group",
			expectedStream: "test-stream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := arn.ParseStreamArn(tt.arnStr)
			
			if result.GroupName != tt.expectedGroup {
				t.Errorf("Expected group %q, got %q", tt.expectedGroup, result.GroupName)
			}
			
			if result.StreamName != tt.expectedStream {
				t.Errorf("Expected stream %q, got %q", tt.expectedStream, result.StreamName)
			}
		})
	}
}
