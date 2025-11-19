package model

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

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
var AwsProfile string

// creates message for refreshing log groups
func getLogGroups() tea.Msg {
	client, err := fetch.CreateClient(AwsProfile)
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
	client, err := fetch.CreateClient(AwsProfile)
	if err != nil {
		return errMsg{err}
	}
	logStreams, err := fetch.FetchLogStreams(client, logGroupName, 1000) // TODO: max results
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
	client, err := fetch.CreateClient(AwsProfile)
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

	groupsList  *GroupsList
	streamsList *StreamsList
	eventsViewer *EventsViewer
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
		m.groupsList.SetGroups(msg)
		m.logGroups = msg
	case logStreamMsg:
		m.streamsList.SetStreams(msg.groupName, msg.streams)
		m.logStreams = msg.streams
	case logEventMsg:
		m.eventsViewer.SetEvents(msg.events)
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.groupsList.SetSize(msg.Width-h, msg.Height-v)
		m.streamsList.SetSize(msg.Width-h, msg.Height-v)
		m.eventsViewer.SetSize(msg.Width-h, msg.Height-v)
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
			if !m.groupsList.Model.SettingFilter() {
				m.mode = Streams
				return m, func() tea.Msg {
					return getLogStreams(
						m.groupsList.Model.SelectedItem().(item).Title(),
					)
				}
			}
		}
	}
	var cmd tea.Cmd
	comp, cmd := m.groupsList.Update(msg)
	m.groupsList = comp.(*GroupsList)
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
			if !m.streamsList.Model.SettingFilter() {
				m.mode = Page
				return m, func() tea.Msg {
					return getLogEvents(
						m.groupsList.Model.SelectedItem().(item).Title(),
						m.streamsList.Model.SelectedItem().(item).Title(),
					)
				}
			}
		case "esc":
			m.mode = Groups
			return m, nil
		}
	}

	var cmd tea.Cmd
	comp, cmd := m.streamsList.Update(msg)
	m.streamsList = comp.(*StreamsList)
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
			m.eventsViewer.SetContent("")
			return m, nil
		}
	}
	var cmd tea.Cmd
	comp, cmd := m.eventsViewer.Update(msg)
	m.eventsViewer = comp.(*EventsViewer)
	return m, cmd
}

// primary rendering function
func (m model) View() string {
	if m.mode == Groups {
		return docStyle.Render(m.groupsList.View())
	} else if m.mode == Streams {
		return docStyle.Render(m.streamsList.View())
	} else {
		return docStyle.Render(m.eventsViewer.View())
	}
}

// initialize model data
func InitialModel() model {
	return model{
		groupsList:   NewGroupsList(),
		streamsList:  NewStreamsList(),
		eventsViewer: NewEventsViewer(),
	}
}
