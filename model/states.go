package model

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/derricw/cwl/fetch"
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
	case errMsg:
		m.groupsList.Model.Title = "Log Groups - " + m.config.Styles.ErrorStyle.Render("Error: "+msg.Error())
		return s, nil
	case tea.KeyMsg:
		switch msg.String() {
		case m.config.KeyBinds.Quit:
			return s, tea.Quit
		case m.config.KeyBinds.Back:
			return s, nil
		case m.config.KeyBinds.Select:
			if !m.groupsList.Model.SettingFilter() {
				if selected := m.groupsList.Model.SelectedItem(); selected != nil {
					groupName := selected.(item).Title()
					m.currentGroupName = groupName
					m.logStreams = nil
					m.streamFetchID++
					return &StreamsState{}, NewLoadStreamsAction(m.deps, groupName, m.streamFetchID).Execute()
				}
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

// checkPreview fires a preview fetch if the selected stream changed
func (s *StreamsState) checkPreview(m *model) tea.Cmd {
	if streamItem := m.streamsList.Model.SelectedItem(); streamItem != nil {
		name := streamItem.(item).Title()
		if name != m.previewStream {
			m.previewStream = name
			m.previewContent = ""
			m.previewFetchID++
			return NewLoadPreviewEventsAction(m.deps, m.currentGroupName, name, m.previewFetchID).Execute()
		}
	}
	return nil
}

func (s *StreamsState) Update(msg tea.Msg, m *model) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.streamsList.Model.Title = "Log Streams - " + m.config.Styles.ErrorStyle.Render("Error: "+msg.Error())
		return s, nil
	case saveLogsMsg:
		m.streamSaving = false
		if msg.err != nil {
			m.streamSaveStatus = "Save failed: " + msg.err.Error()
		} else {
			m.streamSaveStatus = "Saved to " + msg.path
		}
		cmd := m.streamsList.Model.NewStatusMessage(m.config.Styles.FooterStyle.Render(m.streamSaveStatus))
		return s, cmd
	case logStreamPartialMsg:
		cmds := []tea.Cmd{msg.nextCmd}
		if previewCmd := s.checkPreview(m); previewCmd != nil {
			cmds = append(cmds, previewCmd)
		}
		return s, tea.Batch(cmds...)
	case tickMsg:
		// periodically check for new streams
		m.logStreams = nil
		m.streamFetchID++
		return s, tea.Batch(NewLoadStreamsAction(m.deps, m.currentGroupName, m.streamFetchID).Execute(), s.tickCmd())
	case tea.KeyMsg:
		switch msg.String() {
		case m.config.KeyBinds.Quit:
			return s, tea.Quit
		case "esc":
			if m.streamsList.Model.SettingFilter() || m.streamsList.Model.FilterState() == list.FilterApplied {
				m.streamsList.Model.ResetFilter()
				return s, nil
			}
			if m.initialGroup != "" {
				return s, tea.Quit
			}
			return &GroupsState{}, nil
		case m.config.KeyBinds.Select:
			if !m.streamsList.Model.SettingFilter() {
				if streamItem := m.streamsList.Model.SelectedItem(); streamItem != nil {
					streamName := streamItem.(item).Title()
					return &EventsState{}, NewLoadEventsAction(m.deps, m.currentGroupName, streamName).Execute()
				}
			}
		case m.config.KeyBinds.Back:
			if !m.streamsList.Model.SettingFilter() && m.streamsList.Model.FilterState() != list.FilterApplied {
				if m.initialGroup != "" {
					return s, tea.Quit
				}
				return &GroupsState{}, nil
			}
		case "p":
			if !m.streamsList.Model.SettingFilter() {
				m.previewEnabled = !m.previewEnabled
				return s, nil
			}
		case m.config.KeyBinds.SaveLogs:
			if !m.streamsList.Model.SettingFilter() && !m.streamSaving {
				if streamItem := m.streamsList.Model.SelectedItem(); streamItem != nil {
					streamName := streamItem.(item).Title()
					m.streamSaving = true
					m.streamSaveStatus = "Saving " + streamName + "..."
					statusCmd := m.streamsList.Model.NewStatusMessage(m.config.Styles.FooterStyle.Render(m.streamSaveStatus))
					return s, tea.Batch(statusCmd, saveStreamCmd(m.deps, m.currentGroupName, streamName))
				}
			}
		}
	}

	comp, cmd := m.streamsList.Update(msg)
	m.streamsList = comp.(*StreamsList)
	previewCmd := s.checkPreview(m)
	if previewCmd != nil {
		if cmd != nil {
			return s, tea.Batch(cmd, previewCmd)
		}
		return s, previewCmd
	}
	return s, cmd
}

func (s *StreamsState) View(m *model) string {
	if m.previewStream == "" || m.termWidth < 100 || !m.previewEnabled {
		return m.config.Styles.DocStyle.Render(m.streamsList.View())
	}
	fullWidth := m.streamsList.Model.Width()
	listWidth := fullWidth / 2
	previewWidth := fullWidth - listWidth - 4 // 2 for gap, 2 for border
	m.streamsList.Model.SetWidth(listWidth)
	listView := m.streamsList.View()
	m.streamsList.Model.SetWidth(fullWidth)
	listHeight := lipgloss.Height(listView)
	previewHeader := m.config.Styles.HeaderStyle.Render("Preview: " + m.previewStream)
	innerContent := lipgloss.NewStyle().
		Width(previewWidth).
		Height(listHeight - 2).
		MaxHeight(listHeight - 2).
		Render(previewHeader + "\n" + m.previewContent)
	preview := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Render(innerContent)
	return m.config.Styles.DocStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Top, listView, "  ", preview),
	)
}

func (s *StreamsState) Enter(m *model) tea.Cmd {
	return s.tickCmd()
}

func (s *StreamsState) Exit(m *model) {
	m.previewStream = ""
	m.previewContent = ""
}

// background ticker (used to refresh stream list)
func (s *StreamsState) tickCmd() tea.Cmd {
	return tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// EventsState handles log events view
type EventsState struct {
	groupName  string
	streamName string
	saveStatus string
}

func (s *EventsState) Update(msg tea.Msg, m *model) (State, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case errMsg:
		m.errorText = msg.Error()
		return s, nil
	case saveLogsMsg:
		if msg.err != nil {
			s.saveStatus = "Save failed: " + msg.err.Error()
		} else {
			s.saveStatus = "Saved to " + msg.path
		}
		return s, nil
	case tickMsg:
		if !m.eventsViewer.IsLoading() {
			lastTime := m.eventsViewer.GetLastEventTime()
			cmds = append(cmds, NewPollNewEventsAction(m.deps, s.groupName, s.streamName, lastTime).Execute())
		}
		return s, tea.Batch(append(cmds, s.tickCmd())...)
	case tea.KeyMsg:
		if m.eventsViewer.IsFiltering() {
			switch msg.String() {
			case "esc":
				m.eventsViewer.ClearFilter()
				m.eventsViewer.StopFiltering()
				m.eventsViewer.RefreshContentWithTimestamps(m.wrapEnabled, m.showTimestamps)
				return s, nil
			case "enter":
				m.eventsViewer.StopFiltering()
				return s, nil
			}
		} else {
			switch msg.String() {
			case m.config.KeyBinds.Quit:
				return s, tea.Quit
			case m.config.KeyBinds.Back:
				m.eventsViewer.SetContent("")
				m.eventsViewer.lastEventTime = nil
				return &StreamsState{}, nil
			}
			if !m.eventsViewer.loading {
				switch msg.String() {
			case m.config.KeyBinds.ScrollBottom:
				m.eventsViewer.GotoBottom()
				return s, nil
			case m.config.KeyBinds.ScrollTop:
				m.eventsViewer.GotoTop()
				return s, nil
			case m.config.KeyBinds.ToggleWrap:
				m.wrapEnabled = !m.wrapEnabled
				m.eventsViewer.RefreshContentWithTimestamps(m.wrapEnabled, m.showTimestamps)
				return s, nil
			case "t":
				m.showTimestamps = !m.showTimestamps
				m.eventsViewer.RefreshContentWithTimestamps(m.wrapEnabled, m.showTimestamps)
				return s, nil
			case m.config.KeyBinds.Filter:
				m.eventsViewer.StartFiltering()
				return s, nil
			case m.config.KeyBinds.SaveLogs:
				s.saveStatus = "Saving..."
				return s, saveLogsCmd(m.eventsViewer.rawEvents)
				}
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
		m.eventsViewer.RefreshContentWithTimestamps(m.wrapEnabled, m.showTimestamps)
	}

	return s, tea.Batch(cmds...)
}

func (s *EventsState) View(m *model) string {
	header := m.config.Styles.HeaderStyle.Render("Log Stream: " + m.currentStreamName)
	footerText := fmt.Sprintf("%.0f%%", m.eventsViewer.ScrollPercent()*100)
	if m.eventsViewer.IsPaginating() {
		footerText += " " + m.eventsViewer.SpinnerView() + " loading events..."
	}
	if m.eventsViewer.IsFiltering() {
		footerText += " | ESC/Enter to exit filter"
	} else {
		footerText += " | / to filter | t timestamps | w wrap | s save"
	}
	if s.saveStatus != "" {
		footerText += " | " + s.saveStatus
	}
	footer := m.config.Styles.FooterStyle.Render(footerText)
	content := m.eventsViewer.View()
	if m.errorText != "" {
		errLine := m.config.Styles.ErrorStyle.Render("Error: " + m.errorText)
		return m.config.Styles.DocStyle.Render(header + "\n" + content + "\n" + footer + "\n" + errLine)
	}
	return m.config.Styles.DocStyle.Render(header + "\n" + content + "\n" + footer)
}

func (s *EventsState) Enter(m *model) tea.Cmd {
	m.errorText = ""
	s.groupName = m.currentGroupName
	if streamItem := m.streamsList.Model.SelectedItem(); streamItem != nil {
		s.streamName = streamItem.(item).Title()
	}
	return s.tickCmd()
}

func (s *EventsState) tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (s *EventsState) Exit(m *model) {
}

func saveLogsCmd(events []types.OutputLogEvent) tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return saveLogsMsg{err: err}
		}
		dir := filepath.Join(home, "Downloads", "cwl", "logs")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return saveLogsMsg{err: err}
		}
		filename := time.Now().Format("2006-01-02T15-04-05") + ".log"
		path := filepath.Join(dir, filename)
		var sb strings.Builder
		for _, e := range events {
			if e.Message != nil {
				sb.WriteString(strings.TrimRight(*e.Message, "\r\n"))
				sb.WriteByte('\n')
			}
		}
		if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
			return saveLogsMsg{err: err}
		}
		return saveLogsMsg{path: path}
	}
}

func saveStreamCmd(deps *Dependencies, groupName, streamName string) tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return saveLogsMsg{err: err}
		}
		dir := filepath.Join(home, "Downloads", "cwl", "logs")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return saveLogsMsg{err: err}
		}
		filename := time.Now().Format("2006-01-02T15-04-05") + ".log"
		path := filepath.Join(dir, filename)
		f, err := os.Create(path)
		if err != nil {
			return saveLogsMsg{err: err}
		}
		defer f.Close()
		err = fetch.FetchLogEventsStreaming(deps.Client, groupName, streamName, func(events []types.OutputLogEvent) error {
			for _, e := range events {
				if e.Message != nil {
					msg := strings.TrimRight(*e.Message, "\r\n")
					if _, err := f.WriteString(msg + "\n"); err != nil {
						return err
					}
				}
			}
			return nil
		})
		if err != nil {
			return saveLogsMsg{err: err}
		}
		return saveLogsMsg{path: path}
	}
}
