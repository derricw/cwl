package model

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/derricw/cwl/provider"
)

// Action interface for command pattern
type Action interface {
	Execute() tea.Cmd
}

// LoadGroupsAction loads log groups
type LoadGroupsAction struct {
	deps *Dependencies
}

func NewLoadGroupsAction(deps *Dependencies) *LoadGroupsAction {
	return &LoadGroupsAction{deps: deps}
}

func (a *LoadGroupsAction) Execute() tea.Cmd {
	return func() tea.Msg {
		logGroups, err := a.deps.Backend.FetchGroups("")
		if err != nil {
			return errMsg{err}
		}
		return logGroupMsg(logGroups)
	}
}

// LoadStreamsAction loads log streams for a group
type LoadStreamsAction struct {
	deps      *Dependencies
	groupName string
	fetchID   int
}

func NewLoadStreamsAction(deps *Dependencies, groupName string, fetchID int) *LoadStreamsAction {
	return &LoadStreamsAction{
		deps:      deps,
		groupName: groupName,
		fetchID:   fetchID,
	}
}

var errMaxStreamsReached = fmt.Errorf("max streams reached")

func (a *LoadStreamsAction) Execute() tea.Cmd {
	ch := make(chan tea.Msg, 100)
	go func() {
		defer close(ch)
		count := 0
		err := a.deps.Backend.FetchStreamsStreaming(a.groupName, func(streams []provider.LogStream) error {
			count += len(streams)
			ch <- logStreamPartialMsg{groupName: a.groupName, streams: streams, fetchID: a.fetchID}
			if count >= 20000 {
				return errMaxStreamsReached
			}
			return nil
		})
		if err != nil && err != errMaxStreamsReached {
			ch <- errMsg{err}
		}
	}()
	return waitForStreamBatch(ch)
}

func waitForStreamBatch(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		select {
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if partial, ok := msg.(logStreamPartialMsg); ok {
				partial.nextCmd = waitForStreamBatch(ch)
				return partial
			}
			return msg
		case <-time.After(50 * time.Millisecond):
			return waitForStreamBatch(ch)()
		}
	}
}

// LoadEventsAction loads log events for a stream
type LoadEventsAction struct {
	deps       *Dependencies
	groupName  string
	streamName string
}

func NewLoadEventsAction(deps *Dependencies, groupName, streamName string) *LoadEventsAction {
	return &LoadEventsAction{
		deps:       deps,
		groupName:  groupName,
		streamName: streamName,
	}
}

func (a *LoadEventsAction) Execute() tea.Cmd {
	return func() tea.Msg {
		return logEventMsg{
			groupName:  a.groupName,
			streamName: a.streamName,
			events:     []provider.LogEvent{},
		}
	}
}

type LoadEventsStreamingAction struct {
	deps       *Dependencies
	groupName  string
	streamName string
}

func NewLoadEventsStreamingAction(deps *Dependencies, groupName, streamName string) *LoadEventsStreamingAction {
	return &LoadEventsStreamingAction{
		deps:       deps,
		groupName:  groupName,
		streamName: streamName,
	}
}

func (a *LoadEventsStreamingAction) Execute() tea.Cmd {
	ch := make(chan tea.Msg, 100)
	go func() {
		defer close(ch)
		err := a.deps.Backend.FetchEventsStreaming(a.groupName, a.streamName, func(events []provider.LogEvent) error {
			ch <- logEventPartialMsg{events: events}
			return nil
		})
		if err != nil {
			ch <- errMsg{err}
		}
	}()
	return waitForBatch(ch)
}

func waitForBatch(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		select {
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if partial, ok := msg.(logEventPartialMsg); ok {
				partial.nextCmd = waitForBatch(ch)
				return partial
			}
			return msg
		case <-time.After(50 * time.Millisecond):
			return waitForBatch(ch)()
		}
	}
}

// LoadPreviewEventsAction fetches the last N events for the preview pane
type LoadPreviewEventsAction struct {
	deps       *Dependencies
	groupName  string
	streamName string
	fetchID    int
}

func NewLoadPreviewEventsAction(deps *Dependencies, groupName, streamName string, fetchID int) *LoadPreviewEventsAction {
	return &LoadPreviewEventsAction{deps: deps, groupName: groupName, streamName: streamName, fetchID: fetchID}
}

func (a *LoadPreviewEventsAction) Execute() tea.Cmd {
	return func() tea.Msg {
		events, err := a.deps.Backend.FetchLastEvents(a.groupName, a.streamName, 20)
		if err != nil {
			return nil
		}
		return previewEventsMsg{streamName: a.streamName, events: events, fetchID: a.fetchID}
	}
}

type PollNewEventsAction struct {
	deps       *Dependencies
	groupName  string
	streamName string
	startTime  *time.Time
}

func NewPollNewEventsAction(deps *Dependencies, groupName, streamName string, startTime *time.Time) *PollNewEventsAction {
	return &PollNewEventsAction{
		deps:       deps,
		groupName:  groupName,
		streamName: streamName,
		startTime:  startTime,
	}
}

func (a *PollNewEventsAction) Execute() tea.Cmd {
	return func() tea.Msg {
		events, err := a.deps.Backend.FetchNewEvents(a.groupName, a.streamName, a.startTime)
		if err != nil {
			return errMsg{err}
		}
		return newEventsMsg(events)
	}
}