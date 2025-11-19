package model

import (
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
		events, err := fetch.FetchLogEvents(a.deps.Client, a.groupName, a.streamName)
		if err != nil {
			return errMsg{err}
		}
		return logEventMsg{
			groupName:  a.groupName,
			streamName: a.streamName,
			events:     events,
		}
	}
}