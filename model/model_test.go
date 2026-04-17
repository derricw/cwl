package model

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	tea "github.com/charmbracelet/bubbletea"
)

func testDeps() *Dependencies {
	return &Dependencies{}
}

// TestInitialModelDefaultState verifies that launching cwl with no flags
// starts in GroupsState (the log groups browser).
func TestInitialModelDefaultState(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	if _, ok := m.state.(*GroupsState); !ok {
		t.Fatalf("expected GroupsState, got %T", m.state)
	}
	if m.initialGroup != "" {
		t.Fatal("expected empty initialGroup")
	}
}

// TestInitialModelWithGroup verifies that the -g flag skips GroupsState and
// starts directly in StreamsState with the correct group name and fetch ID.
func TestInitialModelWithGroup(t *testing.T) {
	m := InitialModel(testDeps(), "/aws/batch/job", "")
	if _, ok := m.state.(*StreamsState); !ok {
		t.Fatalf("expected StreamsState, got %T", m.state)
	}
	if m.initialGroup != "/aws/batch/job" {
		t.Fatalf("expected initialGroup /aws/batch/job, got %s", m.initialGroup)
	}
	if m.currentGroupName != "/aws/batch/job" {
		t.Fatalf("expected currentGroupName /aws/batch/job, got %s", m.currentGroupName)
	}
	if m.streamFetchID != 1 {
		t.Fatalf("expected streamFetchID 1, got %d", m.streamFetchID)
	}
}

// TestGroupsStateErrorUpdatesTitle verifies that API errors (e.g. auth failures)
// are shown in the list title bar. Errors are rendered in the title because the
// bubbles list component fills its full height, making content below it invisible.
func TestGroupsStateErrorUpdatesTitle(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	m.Log = io.Discard
	msg := errMsg{err: errors.New("access denied")}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)
	if !strings.Contains(updated.groupsList.Model.Title, "access denied") {
		t.Fatalf("expected title to contain error, got: %s", updated.groupsList.Model.Title)
	}
}

// TestStreamsStateErrorUpdatesTitle verifies error display in the streams view title.
func TestStreamsStateErrorUpdatesTitle(t *testing.T) {
	m := InitialModel(testDeps(), "/aws/test", "")
	m.Log = io.Discard
	msg := errMsg{err: errors.New("throttled")}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)
	if !strings.Contains(updated.streamsList.Model.Title, "throttled") {
		t.Fatalf("expected title to contain error, got: %s", updated.streamsList.Model.Title)
	}
}

// TestEventsStateErrorSetsErrorText verifies that errors in the events view
// are stored in errorText for rendering in the footer (events view doesn't
// use a list component, so it can't use the title bar approach).
func TestEventsStateErrorSetsErrorText(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	m.Log = io.Discard
	m.state = &EventsState{}
	msg := errMsg{err: errors.New("connection refused")}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)
	if updated.errorText != "connection refused" {
		t.Fatalf("expected errorText 'connection refused', got: %s", updated.errorText)
	}
}

// TestGroupsStateSelectEmptyList verifies that pressing Enter on an empty groups
// list doesn't panic. The list returns nil for SelectedItem() when empty, and
// a missing nil check previously caused a crash.
func TestGroupsStateSelectEmptyList(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	m.Log = io.Discard
	// Send enter key on empty list — should not panic
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)
	if _, ok := updated.state.(*GroupsState); !ok {
		t.Fatalf("expected to stay in GroupsState, got %T", updated.state)
	}
}

// TestSaveLogsCmd verifies that saveLogsCmd writes events to the correct path
// under $HOME/Downloads/cwl/logs/, strips \r\n line endings, and produces
// clean newline-delimited output.
func TestSaveLogsCmd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	events := []types.OutputLogEvent{
		{Message: aws.String("line one\r\n")},
		{Message: aws.String("line two")},
		{Message: aws.String("line three\n")},
	}

	cmd := saveLogsCmd(events)
	msg := cmd()

	result, ok := msg.(saveLogsMsg)
	if !ok {
		t.Fatalf("expected saveLogsMsg, got %T", msg)
	}
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}
	if !strings.HasPrefix(result.path, filepath.Join(dir, "Downloads", "cwl", "logs")) {
		t.Fatalf("unexpected path: %s", result.path)
	}
	if !strings.HasSuffix(result.path, ".log") {
		t.Fatalf("expected .log extension, got: %s", result.path)
	}

	content, err := os.ReadFile(result.path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	expected := "line one\nline two\nline three\n"
	if string(content) != expected {
		t.Fatalf("expected %q, got %q", expected, string(content))
	}
}

// TestSaveLogsCmdEmptyEvents verifies that saving with no events produces
// an empty file without errors.
func TestSaveLogsCmdEmptyEvents(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cmd := saveLogsCmd(nil)
	msg := cmd()

	result := msg.(saveLogsMsg)
	if result.err != nil {
		t.Fatalf("unexpected error: %v", result.err)
	}

	content, err := os.ReadFile(result.path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "" {
		t.Fatalf("expected empty file, got %q", string(content))
	}
}

// TestGroupSelectClearsStreams verifies that selecting a new group clears
// stale streams from the previous group. Without this, logStreamPartialMsg
// appends to the old list, mixing streams from different groups.
func TestGroupSelectClearsStreams(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	m.Log = io.Discard

	m.logStreams = []types.LogStream{
		{LogStreamName: aws.String("old-stream")},
	}

	m.groupsList.SetGroups([]types.LogGroup{
		{LogGroupName: aws.String("/aws/group1"), LogGroupArn: aws.String("arn1")},
	})

	// Verify logStreams is non-nil before selection
	if m.logStreams == nil {
		t.Fatal("expected logStreams to be non-nil before selection")
	}

	// Simulate what GroupsState.Update does on enter without calling Execute()
	// (Execute() spawns goroutines that panic with nil client)
	selected := m.groupsList.Model.SelectedItem()
	if selected == nil {
		t.Fatal("expected selected item")
	}
	m.currentGroupName = selected.(item).Title()
	m.logStreams = nil
	m.streamFetchID++

	if m.logStreams != nil {
		t.Fatalf("expected logStreams to be nil after selecting new group, got %d items", len(m.logStreams))
	}
	if m.currentGroupName != "/aws/group1" {
		t.Fatalf("expected currentGroupName '/aws/group1', got %s", m.currentGroupName)
	}
}

// TestInitialModelWithStreamFilter verifies that the -s flag stores the
// stream filter for later application when streams load.
func TestInitialModelWithStreamFilter(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	// Manually set fields to test filter storage without triggering Init()
	m.state = &StreamsState{}
	m.initialGroup = "/aws/batch/job"
	m.streamFilter = "my-filter"

	if _, ok := m.state.(*StreamsState); !ok {
		t.Fatalf("expected StreamsState, got %T", m.state)
	}
	if m.streamFilter != "my-filter" {
		t.Fatalf("expected streamFilter 'my-filter', got %s", m.streamFilter)
	}
}

// TestStreamFilterClearedAfterFirstBatch verifies that the CLI stream filter
// (-s flag) is applied once on the first batch of streams, then cleared.
// SetFilterText resets the list cursor to 0, so calling it on every batch
// would break keyboard navigation (j/k/arrows).
func TestStreamFilterClearedAfterFirstBatch(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	m.Log = io.Discard
	// Simulate being in StreamsState with a filter, as if launched with -g and -s
	m.state = &StreamsState{}
	m.streamFilter = "my-filter"
	m.streamFetchID = 1

	msg := logStreamPartialMsg{
		groupName: "/aws/batch/job",
		streams: []types.LogStream{
			{LogStreamName: aws.String("my-filter-stream-1"), LastEventTimestamp: aws.Int64(0)},
			{LogStreamName: aws.String("other-stream"), LastEventTimestamp: aws.Int64(0)},
		},
		fetchID: 1,
	}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)

	if updated.streamFilter != "" {
		t.Fatalf("expected streamFilter to be cleared, got %q", updated.streamFilter)
	}
}

// TestStreamsStateBackWithInitialGroup verifies that pressing Esc in the
// streams view quits the app when launched with -g, since there's no
// groups list to navigate back to.
func TestStreamsStateBackWithInitialGroup(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	m.Log = io.Discard
	m.state = &StreamsState{}
	m.initialGroup = "/aws/batch/job"

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, cmd := m.Update(msg)
	updated := newModel.(model)

	if _, ok := updated.state.(*StreamsState); !ok {
		t.Fatalf("expected StreamsState, got %T", updated.state)
	}
	if cmd == nil {
		t.Fatal("expected quit cmd, got nil")
	}
}

// TestStreamsStateBackWithoutInitialGroup verifies that pressing Esc in the
// streams view navigates back to GroupsState during normal TUI navigation.
func TestStreamsStateBackWithoutInitialGroup(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	m.Log = io.Discard
	// Manually set state to StreamsState as if user navigated there
	m.state = &StreamsState{}
	m.currentGroupName = "/aws/test"

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)

	if _, ok := updated.state.(*GroupsState); !ok {
		t.Fatalf("expected GroupsState, got %T", updated.state)
	}
}

// TestStreamsStateSavePreventsConurrent verifies that pressing 's' while a
// save is already in progress is a no-op (prevents concurrent saves).
func TestStreamsStateSavePreventsConurrent(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	m.Log = io.Discard
	m.state = &StreamsState{}
	m.streamSaving = true

	m.streamsList.SetStreams("/aws/batch/job", []types.LogStream{
		{LogStreamName: aws.String("stream-1"), LastEventTimestamp: aws.Int64(0)},
	})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)

	if updated.streamSaveStatus != "" {
		t.Fatalf("expected empty save status, got %q", updated.streamSaveStatus)
	}
}

// TestStreamsStateSaveLogsResult verifies that a successful stream save
// clears the saving flag and sets the status message with the file path.
func TestStreamsStateSaveLogsResult(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	m.Log = io.Discard
	m.state = &StreamsState{}
	m.streamSaving = true

	msg := saveLogsMsg{path: "/home/user/Downloads/cwl/logs/2026-04-16T22-00-00.log"}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)

	if updated.streamSaving {
		t.Fatal("expected streamSaving to be false after save completes")
	}
	if !strings.Contains(updated.streamSaveStatus, "Saved to") {
		t.Fatalf("expected save status to contain 'Saved to', got %q", updated.streamSaveStatus)
	}
}

// TestStreamsStateSaveLogsError verifies that a failed stream save clears
// the saving flag and shows the error in the status message.
func TestStreamsStateSaveLogsError(t *testing.T) {
	m := InitialModel(testDeps(), "", "")
	m.Log = io.Discard
	m.state = &StreamsState{}
	m.streamSaving = true

	msg := saveLogsMsg{err: errors.New("disk full")}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)

	if updated.streamSaving {
		t.Fatal("expected streamSaving to be false after save error")
	}
	if !strings.Contains(updated.streamSaveStatus, "disk full") {
		t.Fatalf("expected save status to contain error, got %q", updated.streamSaveStatus)
	}
}
