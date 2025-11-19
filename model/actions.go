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
	profile string
}

func NewLoadGroupsAction(profile string) *LoadGroupsAction {
	return &LoadGroupsAction{profile: profile}
}

func (a *LoadGroupsAction) Execute() tea.Cmd {
	return func() tea.Msg {
		client, err := fetch.CreateClient(a.profile)
		if err != nil {
			return errMsg{err}
		}
		
		logGroups, err := fetch.FetchLogGroups(client)
		if err != nil {
			return errMsg{err}
		}
		return logGroupMsg(logGroups)
	}
}

// LoadStreamsAction loads log streams for a group
type LoadStreamsAction struct {
	profile   string
	groupName string
}

func NewLoadStreamsAction(profile, groupName string) *LoadStreamsAction {
	return &LoadStreamsAction{
		profile:   profile,
		groupName: groupName,
	}
}

func (a *LoadStreamsAction) Execute() tea.Cmd {
	return func() tea.Msg {
		client, err := fetch.CreateClient(a.profile)
		if err != nil {
			return errMsg{err}
		}
		
		logStreams, err := fetch.FetchLogStreams(client, a.groupName, 1000)
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
	profile    string
	groupName  string
	streamName string
}

func NewLoadEventsAction(profile, groupName, streamName string) *LoadEventsAction {
	return &LoadEventsAction{
		profile:    profile,
		groupName:  groupName,
		streamName: streamName,
	}
}

func (a *LoadEventsAction) Execute() tea.Cmd {
	return func() tea.Msg {
		client, err := fetch.CreateClient(a.profile)
		if err != nil {
			return errMsg{err}
		}
		
		events, err := fetch.FetchLogEvents(client, a.groupName, a.streamName)
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