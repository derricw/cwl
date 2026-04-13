package cmd

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type mockEventsClient struct {
	pages []mockPage
	call  int
}

type mockPage struct {
	events []types.OutputLogEvent
	token  string
}

func (m *mockEventsClient) GetLogEvents(ctx context.Context, params *cloudwatchlogs.GetLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetLogEventsOutput, error) {
	if m.call >= len(m.pages) {
		// Return same token to signal end
		lastToken := m.pages[len(m.pages)-1].token
		return &cloudwatchlogs.GetLogEventsOutput{
			Events:            nil,
			NextForwardToken:  &lastToken,
		}, nil
	}
	page := m.pages[m.call]
	m.call++
	return &cloudwatchlogs.GetLogEventsOutput{
		Events:           page.events,
		NextForwardToken: &page.token,
	}, nil
}

func (m *mockEventsClient) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	return nil, nil
}
func (m *mockEventsClient) DescribeLogStreams(ctx context.Context, params *cloudwatchlogs.DescribeLogStreamsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	return nil, nil
}
func (m *mockEventsClient) CreateLogStream(ctx context.Context, params *cloudwatchlogs.CreateLogStreamInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.CreateLogStreamOutput, error) {
	return nil, nil
}
func (m *mockEventsClient) PutLogEvents(ctx context.Context, params *cloudwatchlogs.PutLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.PutLogEventsOutput, error) {
	return nil, nil
}
func (m *mockEventsClient) StartQuery(ctx context.Context, params *cloudwatchlogs.StartQueryInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.StartQueryOutput, error) {
	return nil, nil
}
func (m *mockEventsClient) GetQueryResults(ctx context.Context, params *cloudwatchlogs.GetQueryResultsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.GetQueryResultsOutput, error) {
	return nil, nil
}

func makeEvents(n int) []types.OutputLogEvent {
	events := make([]types.OutputLogEvent, n)
	for i := range events {
		events[i] = types.OutputLogEvent{Message: aws.String("msg")}
	}
	return events
}

func collectEvents(ch chan Event) []Event {
	var result []Event
	for e := range ch {
		result = append(result, e)
	}
	return result
}

func TestRequestEventsLimit(t *testing.T) {
	client := &mockEventsClient{
		pages: []mockPage{
			{events: makeEvents(100), token: "t1"},
			{events: makeEvents(100), token: "t2"},
			{events: makeEvents(100), token: "t3"},
		},
	}

	ch := make(chan Event, 10000)
	oldFollow := follow
	follow = false
	defer func() { follow = oldFollow }()

	err := requestEvents(client, "group", "stream", ch, nil, 150)
	close(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := collectEvents(ch)
	if len(got) != 150 {
		t.Fatalf("expected 150 events, got %d", len(got))
	}
}

func TestRequestEventsNoLimit(t *testing.T) {
	client := &mockEventsClient{
		pages: []mockPage{
			{events: makeEvents(50), token: "t1"},
			{events: makeEvents(30), token: "t2"},
		},
	}

	ch := make(chan Event, 10000)
	oldFollow := follow
	follow = false
	defer func() { follow = oldFollow }()

	err := requestEvents(client, "group", "stream", ch, nil, 0)
	close(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := collectEvents(ch)
	if len(got) != 80 {
		t.Fatalf("expected 80 events, got %d", len(got))
	}
}

func TestRequestEventsEmptyPageBailout(t *testing.T) {
	oldFollow := follow
	oldMax := maxEmptyPages
	follow = false
	maxEmptyPages = 3
	defer func() {
		follow = oldFollow
		maxEmptyPages = oldMax
	}()

	// Each page returns 0 events but a different token — should bail after 3
	client := &mockEventsClient{
		pages: []mockPage{
			{events: nil, token: "t1"},
			{events: nil, token: "t2"},
			{events: nil, token: "t3"},
			{events: nil, token: "t4"},
			{events: nil, token: "t5"},
		},
	}

	ch := make(chan Event, 10000)
	err := requestEvents(client, "group", "stream", ch, nil, 0)
	close(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := collectEvents(ch)
	if len(got) != 0 {
		t.Fatalf("expected 0 events, got %d", len(got))
	}
	// Should have stopped after maxEmptyPages (3), not gone through all 5
	if client.call > maxEmptyPages+1 {
		t.Fatalf("expected at most %d API calls, got %d", maxEmptyPages+1, client.call)
	}
}

func TestRequestEventsEmptyPagesResetOnData(t *testing.T) {
	oldFollow := follow
	oldMax := maxEmptyPages
	follow = false
	maxEmptyPages = 2
	defer func() {
		follow = oldFollow
		maxEmptyPages = oldMax
	}()

	// Empty page, then data, then empty pages — counter should reset after data
	client := &mockEventsClient{
		pages: []mockPage{
			{events: nil, token: "t1"},
			{events: makeEvents(5), token: "t2"},
			{events: nil, token: "t3"},
			{events: nil, token: "t4"},
		},
	}

	ch := make(chan Event, 10000)
	err := requestEvents(client, "group", "stream", ch, nil, 0)
	close(ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := collectEvents(ch)
	if len(got) != 5 {
		t.Fatalf("expected 5 events, got %d", len(got))
	}
}
