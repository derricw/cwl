package model

import (
	"fmt"
	"time"

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
		case m.config.KeyBinds.Quit:
			return s, tea.Quit
		case m.config.KeyBinds.Back:
			return s, nil
		case m.config.KeyBinds.Select:
			if !m.groupsList.Model.SettingFilter() {
				groupName := m.groupsList.Model.SelectedItem().(item).Title()
				return &StreamsState{}, NewLoadStreamsAction(m.deps, groupName).Execute()
			}
		}
	}

	comp, cmd := m.groupsList.Update(msg)
	m.groupsList = comp.(*GroupsList)
	return s, cmd
}

func (s *GroupsState) View(m *model) string {
	return m.config.Styles.DocStyle.Render(m.groupsList.View())
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
	case tickMsg:
		// periodically check for new streams
		if selected := m.groupsList.Model.SelectedItem(); selected != nil {
			groupName := selected.(item).Title()
			return s, tea.Batch(NewLoadStreamsAction(m.deps, groupName).Execute(), s.tickCmd())
		}
		return s, s.tickCmd()
	case tea.KeyMsg:
		switch msg.String() {
		case m.config.KeyBinds.Quit:
			return s, tea.Quit
		case m.config.KeyBinds.Select:
			if !m.streamsList.Model.SettingFilter() {
				if groupItem := m.groupsList.Model.SelectedItem(); groupItem != nil {
					if streamItem := m.streamsList.Model.SelectedItem(); streamItem != nil {
						groupName := groupItem.(item).Title()
						streamName := streamItem.(item).Title()
						return &EventsState{}, NewLoadEventsAction(m.deps, groupName, streamName).Execute()
					}
				}
			}
		case m.config.KeyBinds.Back:
			return &GroupsState{}, nil
		}
	}

	comp, cmd := m.streamsList.Update(msg)
	m.streamsList = comp.(*StreamsList)
	return s, cmd
}

func (s *StreamsState) View(m *model) string {
	return m.config.Styles.DocStyle.Render(m.streamsList.View())
}

func (s *StreamsState) Enter(m *model) tea.Cmd {
	return s.tickCmd()
}

func (s *StreamsState) Exit(m *model) {
}

// background ticker (used to refresh stream list)
func (s *StreamsState) tickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// EventsState handles log events view
type EventsState struct{}

func (s *EventsState) Update(msg tea.Msg, m *model) (State, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.eventsViewer.IsFiltering() {
			switch msg.String() {
			case "esc":
				m.eventsViewer.ClearFilter()
				m.eventsViewer.StopFiltering()
				m.eventsViewer.RefreshContent(m.wrapEnabled)
				return s, nil
			case "enter":
				m.eventsViewer.StopFiltering()
				return s, nil
			}
		} else if !m.eventsViewer.loading {
			switch msg.String() {
			case m.config.KeyBinds.Quit:
				return s, tea.Quit
			case m.config.KeyBinds.Back:
				m.eventsViewer.SetContent("")
				return &StreamsState{}, nil
			case m.config.KeyBinds.ScrollBottom:
				m.eventsViewer.GotoBottom()
				return s, nil
			case m.config.KeyBinds.ScrollTop:
				m.eventsViewer.GotoTop()
				return s, nil
			case m.config.KeyBinds.ToggleWrap:
				m.wrapEnabled = !m.wrapEnabled
				m.eventsViewer.RefreshContent(m.wrapEnabled)
				return s, nil
			case m.config.KeyBinds.Filter:
				m.eventsViewer.StartFiltering()
				return s, nil
			}
		}
	}

	comp, cmd := m.eventsViewer.Update(msg)
	m.eventsViewer = comp.(*EventsViewer)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	// Refresh content when filter changes
	if m.eventsViewer.IsFiltering() {
		m.eventsViewer.RefreshContent(m.wrapEnabled)
	}

	return s, tea.Batch(cmds...)
}

func (s *EventsState) View(m *model) string {
	header := m.config.Styles.HeaderStyle.Render("Log Stream: " + m.currentStreamName)
	footerText := fmt.Sprintf("%.0f%%", m.eventsViewer.ScrollPercent()*100)
	if m.eventsViewer.IsFiltering() {
		footerText += " | ESC/Enter to exit filter"
	} else {
		footerText += " | / to filter"
	}
	footer := m.config.Styles.FooterStyle.Render(footerText)
	content := m.eventsViewer.View()
	return m.config.Styles.DocStyle.Render(header + "\n" + content + "\n" + footer)
}

func (s *EventsState) Enter(m *model) tea.Cmd {
	return nil
}

func (s *EventsState) Exit(m *model) {
}
