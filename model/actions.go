package model

import (
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/derricw/cwl/fetch"
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
		logGroups, err := fetch.FetchLogGroups(a.deps.Client)
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
}

func NewLoadStreamsAction(deps *Dependencies, groupName string) *LoadStreamsAction {
	return &LoadStreamsAction{
		deps:      deps,
		groupName: groupName,
	}
}

func (a *LoadStreamsAction) Execute() tea.Cmd {
	return func() tea.Msg {
		logStreams, err := fetch.FetchLogStreams(a.deps.Client, a.groupName, 1000)
		if err != nil {
			return errMsg{err}
		}
		return logStreamMsg{
			groupName: a.groupName,
			streams:   logStreams,
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
			events:     []types.OutputLogEvent{},
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
	return func() tea.Msg {
		ch := make(chan tea.Msg, 10)
		go func() {
			defer close(ch)
			err := fetch.FetchLogEventsStreaming(a.deps.Client, a.groupName, a.streamName, func(events []types.OutputLogEvent) error {
				ch <- logEventPartialMsg{events: events}
				return nil
			})
			if err != nil {
				ch <- errMsg{err}
			}
		}()
		
		msg := <-ch
		if partial, ok := msg.(logEventPartialMsg); ok {
			partial.nextCmd = waitForBatch(ch)
			return partial
		}
		return msg
	}
}

func waitForBatch(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		if partial, ok := msg.(logEventPartialMsg); ok {
			partial.nextCmd = waitForBatch(ch)
			return partial
		}
		return msg
	}
}

type PollNewEventsAction struct {
	deps       *Dependencies
	groupName  string
	streamName string
	startTime  *int64
}

func NewPollNewEventsAction(deps *Dependencies, groupName, streamName string, startTime *int64) *PollNewEventsAction {
	return &PollNewEventsAction{
		deps:       deps,
		groupName:  groupName,
		streamName: streamName,
		startTime:  startTime,
	}
}

func (a *PollNewEventsAction) Execute() tea.Cmd {
	return func() tea.Msg {
		events, err := fetch.FetchNewLogEvents(a.deps.Client, a.groupName, a.streamName, a.startTime)
		if err != nil {
			return errMsg{err}
		}
		return newEventsMsg(events)
	}
}