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

func TestInitialModelDefaultState(t *testing.T) {
	m := InitialModel(testDeps(), "")
	if _, ok := m.state.(*GroupsState); !ok {
		t.Fatalf("expected GroupsState, got %T", m.state)
	}
	if m.initialGroup != "" {
		t.Fatal("expected empty initialGroup")
	}
}

func TestInitialModelWithGroup(t *testing.T) {
	m := InitialModel(testDeps(), "/aws/batch/job")
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

func TestGroupsStateErrorUpdatesTitle(t *testing.T) {
	m := InitialModel(testDeps(), "")
	m.Log = io.Discard
	msg := errMsg{err: errors.New("access denied")}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)
	if !strings.Contains(updated.groupsList.Model.Title, "access denied") {
		t.Fatalf("expected title to contain error, got: %s", updated.groupsList.Model.Title)
	}
}

func TestStreamsStateErrorUpdatesTitle(t *testing.T) {
	m := InitialModel(testDeps(), "/aws/test")
	m.Log = io.Discard
	msg := errMsg{err: errors.New("throttled")}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)
	if !strings.Contains(updated.streamsList.Model.Title, "throttled") {
		t.Fatalf("expected title to contain error, got: %s", updated.streamsList.Model.Title)
	}
}

func TestEventsStateErrorSetsErrorText(t *testing.T) {
	m := InitialModel(testDeps(), "")
	m.Log = io.Discard
	m.state = &EventsState{}
	msg := errMsg{err: errors.New("connection refused")}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)
	if updated.errorText != "connection refused" {
		t.Fatalf("expected errorText 'connection refused', got: %s", updated.errorText)
	}
}

func TestGroupsStateSelectEmptyList(t *testing.T) {
	m := InitialModel(testDeps(), "")
	m.Log = io.Discard
	// Send enter key on empty list — should not panic
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := m.Update(msg)
	updated := newModel.(model)
	if _, ok := updated.state.(*GroupsState); !ok {
		t.Fatalf("expected to stay in GroupsState, got %T", updated.state)
	}
}

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
