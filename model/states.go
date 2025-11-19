package model

import (
	tea "github.com/charmbracelet/bubbletea"
)

// State interface for state machine
type State interface {
	Update(msg tea.Msg, m *model) (State, tea.Cmd)
	View(m *model) string
	Enter(m *model) tea.Cmd
	Exit(m *model)
}

// GroupsState handles log groups view
type GroupsState struct{}

func (s *GroupsState) Update(msg tea.Msg, m *model) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return s, tea.Quit
		case "esc":
			return s, nil
		case "enter":
			if !m.groupsList.Model.SettingFilter() {
				groupName := m.groupsList.Model.SelectedItem().(item).Title()
				return &StreamsState{}, NewLoadStreamsAction(AwsProfile, groupName).Execute()
			}
		}
	}
	
	comp, cmd := m.groupsList.Update(msg)
	m.groupsList = comp.(*GroupsList)
	return s, cmd
}

func (s *GroupsState) View(m *model) string {
	return docStyle.Render(m.groupsList.View())
}

func (s *GroupsState) Enter(m *model) tea.Cmd {
	return nil
}

func (s *GroupsState) Exit(m *model) {
}

// StreamsState handles log streams view
type StreamsState struct{}

func (s *StreamsState) Update(msg tea.Msg, m *model) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return s, tea.Quit
		case "enter":
			if !m.streamsList.Model.SettingFilter() {
				groupName := m.groupsList.Model.SelectedItem().(item).Title()
				streamName := m.streamsList.Model.SelectedItem().(item).Title()
				return &EventsState{}, NewLoadEventsAction(AwsProfile, groupName, streamName).Execute()
			}
		case "esc":
			return &GroupsState{}, nil
		}
	}
	
	comp, cmd := m.streamsList.Update(msg)
	m.streamsList = comp.(*StreamsList)
	return s, cmd
}

func (s *StreamsState) View(m *model) string {
	return docStyle.Render(m.streamsList.View())
}

func (s *StreamsState) Enter(m *model) tea.Cmd {
	return nil
}

func (s *StreamsState) Exit(m *model) {
}

// EventsState handles log events view
type EventsState struct{}

func (s *EventsState) Update(msg tea.Msg, m *model) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return s, tea.Quit
		case "esc":
			m.eventsViewer.SetContent("")
			return &StreamsState{}, nil
		}
	}
	
	comp, cmd := m.eventsViewer.Update(msg)
	m.eventsViewer = comp.(*EventsViewer)
	return s, cmd
}

func (s *EventsState) View(m *model) string {
	return docStyle.Render(m.eventsViewer.View())
}

func (s *EventsState) Enter(m *model) tea.Cmd {
	return nil
}

func (s *EventsState) Exit(m *model) {
}