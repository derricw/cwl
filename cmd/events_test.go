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

func TestEventsArgsValidation(t *testing.T) {
	tests := []struct {
		name        string
		group_      string
		stream_     string
		prefix_     string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid single ARN",
			args:        []string{"arn:aws:logs:us-west-2:123:log-group:g:log-stream:s"},
			expectError: false,
		},
		{
			name:        "valid group and stream flags",
			group_:      "/my/group",
			stream_:     "my-stream",
			expectError: false,
		},
		{
			name:        "valid stdin (no args)",
			expectError: false,
		},
		{
			name:        "too many ARN args",
			args:        []string{"arn1", "arn2"},
			expectError: true,
			errorMsg:    "only one ARN argument expected",
		},
		{
			name:        "group without stream",
			group_:      "/my/group",
			expectError: true,
			errorMsg:    "both --group and --stream must be provided",
		},
		{
			name:        "stream without group",
			stream_:     "my-stream",
			expectError: true,
			errorMsg:    "both --group and --stream must be provided",
		},
		{
			name:        "group+stream with ARN",
			group_:      "/my/group",
			stream_:     "my-stream",
			args:        []string{"arn:aws:logs:us-west-2:123:log-group:g:log-stream:s"},
			expectError: true,
			errorMsg:    "cannot provide ARN when using --group/--stream flags",
		},
		{
			name:        "valid follow-prefix",
			group_:      "/my/group",
			prefix_:     "2025/04/",
			expectError: false,
		},
		{
			name:        "follow-prefix without group",
			prefix_:     "2025/04/",
			expectError: true,
			errorMsg:    "--follow-prefix requires --group",
		},
		{
			name:        "follow-prefix with stream",
			group_:      "/my/group",
			stream_:     "my-stream",
			prefix_:     "2025/04/",
			expectError: true,
			errorMsg:    "--follow-prefix cannot be used with --stream or ARN arguments",
		},
		{
			name:        "follow-prefix with ARN",
			group_:      "/my/group",
			prefix_:     "2025/04/",
			args:        []string{"some-arn"},
			expectError: true,
			errorMsg:    "--follow-prefix cannot be used with --stream or ARN arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set package-level vars
			group = tt.group_
			stream = tt.stream_
			eventsPrefix = tt.prefix_
			defer func() {
				group = ""
				stream = ""
				eventsPrefix = ""
			}()

			err := eventsCmd.Args(eventsCmd, tt.args)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %q", err.Error())
				}
			}
		})
	}
}
