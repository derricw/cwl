package model

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/derricw/cwl/fetch"
)

type mode int

const (
	Groups mode = iota
	Streams
	Page
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

// creates message for refreshing log groups
func getLogGroups() tea.Msg {
	client, err := fetch.CreateClient()
	if err != nil {
		return errMsg{err}
	}

	logGroups, err := fetch.FetchLogGroups(client)
	if err != nil {
		return errMsg{err}
	}
	return logGroupMsg(logGroups)
}

// creates message for refreshing log streams
func getLogStreams(logGroupName string) tea.Msg {
	client, err := fetch.CreateClient()
	if err != nil {
		return errMsg{err}
	}
	logStreams, err := fetch.FetchLogStreams(client, logGroupName)
	if err != nil {
		return errMsg{err}
	}
	return logStreamMsg{
		groupName: logGroupName,
		streams:   logStreams,
	}
}

// creates message for refreshing log events
func getLogEvents(logGroupName, logStreamName string) tea.Msg {
	client, err := fetch.CreateClient()
	if err != nil {
		return errMsg{err}
	}
	events, err := fetch.FetchLogEvents(client, logGroupName, logStreamName)
	if err != nil {
		return errMsg{err}
	}
	return logEventMsg{
		groupName:  logGroupName,
		streamName: logStreamName,
		events:     events,
	}
}

type logGroupMsg []types.LogGroup
type logStreamMsg struct {
	groupName string
	streams   []types.LogStream
}
type logEventMsg struct {
	groupName  string
	streamName string
	events     []types.OutputLogEvent
}

type errMsg struct{ err error }

// For messages that contain errors it's often handy to also implement the
// error interface on the message.
func (e errMsg) Error() string { return e.err.Error() }

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	Log io.Writer

	groupsList  list.Model
	streamsList list.Model
	viewport    viewport.Model
	logGroups   []types.LogGroup
	logStreams  []types.LogStream
	mode        mode
}

func (m model) Init() tea.Cmd {
	return getLogGroups
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case logGroupMsg:
		items := []list.Item{}
		for _, group := range msg {
			items = append(items, item{
				title: *group.LogGroupName,
				desc:  *group.LogGroupArn,
			})
		}
		m.groupsList.SetItems(items)
		m.logGroups = msg
	case logStreamMsg:
		items := []list.Item{}
		for _, stream := range msg.streams {
			m.Log.Write([]byte(fmt.Sprintf("Log Stream: %s\n", *stream.LogStreamName)))
			items = append(items, item{
				title: *stream.LogStreamName,
				desc:  time.Unix(0, *stream.LastEventTimestamp*1000000).Format("2006-01-02 15:04:05"),
			})
		}
		m.streamsList.SetItems(items)
		m.streamsList.Title = fmt.Sprintf("Log Streams: %s", msg.groupName)
		m.logStreams = msg.streams
	case logEventMsg:
		var buffer bytes.Buffer
		for _, event := range msg.events {
			buffer.WriteString(fmt.Sprintf("%s\n", *event.Message))
		}
		result := buffer.String()
		m.viewport.SetContent(result)
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.groupsList.SetSize(msg.Width-h, msg.Height-v)
		m.streamsList.SetSize(msg.Width-h, msg.Height-v)
		m.viewport.Width, m.viewport.Height = msg.Width-h, msg.Height-v
	}
	m.Log.Write([]byte(fmt.Sprintf("%+v\n", msg)))

	switch m.mode {
	case Groups:
		return m.updateGroups(msg)
	case Streams:
		return m.updateStreams(msg)
	case Page:
		return m.updatePage(msg)
	}
	var cmd tea.Cmd
	return m, cmd
}

// process key messages when in Group mode
func (m model) updateGroups(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		switch k {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, nil // don't exit on escape
		case "enter":
			if !m.groupsList.SettingFilter() {
				m.mode = Streams
				return m, func() tea.Msg {
					return getLogStreams(
						m.groupsList.SelectedItem().(item).Title(),
					)
				}
			}
		}
	}
	var cmd tea.Cmd
	m.groupsList, cmd = m.groupsList.Update(msg)
	return m, cmd
}

// process key messages when in Stream mode
func (m model) updateStreams(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		switch k {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if !m.streamsList.SettingFilter() {
				m.mode = Page
				return m, func() tea.Msg {
					return getLogEvents(
						m.groupsList.SelectedItem().(item).Title(),
						m.streamsList.SelectedItem().(item).Title(),
					)
				}
			}
		case "esc":
			m.mode = Groups
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.streamsList, cmd = m.streamsList.Update(msg)
	return m, cmd
}

// process key messages when in Page mode
func (m model) updatePage(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		switch k {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.mode = Streams
			m.viewport.SetContent("")
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// primary rendering function
func (m model) View() string {
	if m.mode == Groups {
		return docStyle.Render(m.groupsList.View())
	} else if m.mode == Streams {
		return docStyle.Render(m.streamsList.View())
	} else {
		return docStyle.Render(m.viewport.View())
	}
}

// initialize model data
func InitialModel() model {
	groups, streams := []list.Item{}, []list.Item{}
	// Delegates definte rendering for list items
	groupsDel := list.NewDefaultDelegate()
	streamsDel := list.NewDefaultDelegate()
	groupsDel.ShowDescription, streamsDel.ShowDescription = false, true
	m := model{
		groupsList:  list.New(groups, groupsDel, 0, 0),
		streamsList: list.New(streams, streamsDel, 0, 0),
		viewport:    viewport.New(50, 50),
	}
	m.groupsList.Title = "Log Groups"
	return m
}
