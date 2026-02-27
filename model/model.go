package model

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type logGroupMsg []types.LogGroup
type logStreamMsg struct {
	groupName string
	streams   []types.LogStream
}
type logStreamPartialMsg struct {
	groupName string
	streams   []types.LogStream
	nextCmd   tea.Cmd
	fetchID   int
}
type logEventMsg struct {
	groupName  string
	streamName string
	events     []types.OutputLogEvent
}
type logEventPartialMsg struct {
	events  []types.OutputLogEvent
	nextCmd tea.Cmd
}
type newEventsMsg []types.OutputLogEvent
type previewEventsMsg struct {
	streamName string
	events     []types.OutputLogEvent
	fetchID    int
}
type tickMsg time.Time

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

	groupsList        *GroupsList
	streamsList       *StreamsList
	eventsViewer      *EventsViewer
	logGroups         []types.LogGroup
	logStreams        []types.LogStream
	state             State
	deps              *Dependencies
	config            *Config
	currentStreamName string
	wrapEnabled       bool
	showTimestamps    bool
	streamFetchID     int
	previewFetchID    int
	previewStream     string
	previewContent    string
	termWidth         int
	termHeight        int
	previewEnabled    bool
}

func (m model) Init() tea.Cmd {
	return NewLoadGroupsAction(m.deps).Execute()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case logGroupMsg:
		m.groupsList.SetGroups(msg)
		m.logGroups = msg
	case logStreamMsg:
		m.streamsList.SetStreams(msg.groupName, msg.streams)
		m.logStreams = msg.streams
	case logStreamPartialMsg:
		if msg.fetchID != m.streamFetchID {
			return m, msg.nextCmd
		}
		if !m.streamsList.Model.SettingFilter() && m.streamsList.Model.FilterState() != list.FilterApplied {
			m.logStreams = append(m.logStreams, msg.streams...)
			m.streamsList.SetStreams(msg.groupName, m.logStreams)
		}
	case logEventMsg:
		m.eventsViewer.SetEvents(msg.events)
		m.currentStreamName = msg.streamName
		spinnerCmd := m.eventsViewer.StartLoading()
		return m, tea.Batch(spinnerCmd, NewLoadEventsStreamingAction(m.deps, msg.groupName, msg.streamName).Execute())
	case logEventPartialMsg:
		m.eventsViewer.AppendEvents(msg.events)
		if msg.nextCmd == nil {
			m.eventsViewer.StopPaginating()
		}
		return m, msg.nextCmd
	case newEventsMsg:
		if len(msg) > 0 {
			m.eventsViewer.AppendEvents(msg)
		}
		return m, nil
	case previewEventsMsg:
		if msg.fetchID == m.previewFetchID {
			m.previewContent = formatPreviewEvents(msg.events)
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		h, v := m.config.Styles.DocStyle.GetFrameSize()
		m.groupsList.SetSize(msg.Width-h, msg.Height-v)
		m.streamsList.SetSize(msg.Width-h, msg.Height-v)
		// Reserve space for header and footer in events viewer
		m.eventsViewer.SetSize(msg.Width-h, msg.Height-v-2)
	}
	m.Log.Write([]byte(fmt.Sprintf("%+v\n", msg)))

	newState, cmd := m.state.Update(msg, &m)
	if newState != m.state {
		m.state.Exit(&m)
		m.state = newState
		enterCmd := m.state.Enter(&m)
		if enterCmd != nil {
			return m, tea.Batch(cmd, enterCmd)
		}
	}
	return m, cmd
}

// primary rendering function
func (m model) View() string {
	return m.state.View(&m)
}

// initialize model data
func InitialModel(deps *Dependencies) model {
	return model{
		groupsList:   NewGroupsList(),
		streamsList:  NewStreamsList(),
		eventsViewer: NewEventsViewer(),
		state:        &GroupsState{},
		deps:         deps,
		config:       DefaultConfig(),
		previewEnabled: true,
	}
}

func formatPreviewEvents(events []types.OutputLogEvent) string {
	var sb strings.Builder
	for _, e := range events {
		if e.Message != nil {
			sb.WriteString(strings.TrimRight(*e.Message, "\n"))
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
