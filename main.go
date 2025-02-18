package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mode int

const (
	Groups mode = iota
	Streams
	Page
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

func getLogGroups() tea.Msg {
	client, err := createClient()
	if err != nil {
		return errMsg{err}
	}

	logGroups, err := fetchLogGroups(client)
	if err != nil {
		return errMsg{err}
	}
	return logGroupMsg(logGroups)
}

func getLogStreams(logGroupName string) tea.Msg {
	client, err := createClient()
	if err != nil {
		return errMsg{err}
	}
	logStreams, err := fetchLogStreams(client, logGroupName)
	if err != nil {
		return errMsg{err}
	}
	return logStreamMsg{
		groupName: logGroupName,
		streams:   logStreams,
	}
}

func getLogEvents(logGroupName, logStreamName string) tea.Msg {
	client, err := createClient()
	if err != nil {
		return errMsg{err}
	}
	events, err := fetchLogEvents(client, logGroupName, logStreamName)
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
	groupsList  list.Model
	streamsList list.Model
	viewport    viewport.Model
	logGroups   []types.LogGroup
	logStreams  []types.LogStream
	mode        mode
	log         io.Writer
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
			m.log.Write([]byte(fmt.Sprintf("Log Stream: %s\n", *stream.LogStreamName)))
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
			buffer.WriteString(*event.Message)
		}
		result := buffer.String()
		m.viewport.SetContent(result)
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.groupsList.SetSize(msg.Width-h, msg.Height-v)
		m.streamsList.SetSize(msg.Width-h, msg.Height-v)
		m.viewport.Width, m.viewport.Height = msg.Width-h, msg.Height-v
	}
	m.log.Write([]byte(fmt.Sprintf("%+v\n", msg)))

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

func (m model) View() string {
	if m.mode == Groups {
		return docStyle.Render(m.groupsList.View())
	} else if m.mode == Streams {
		return docStyle.Render(m.streamsList.View())
	} else {
		return docStyle.Render(m.viewport.View())
	}
}

func initialModel() model {
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

func createClient() (*cloudwatchlogs.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	return cloudwatchlogs.NewFromConfig(cfg), nil
}

func fetchLogGroups(client *cloudwatchlogs.Client) ([]types.LogGroup, error) {
	groups := make([]types.LogGroup, 0)
	var nextToken *string

	for {
		output, err := client.DescribeLogGroups(context.TODO(), &cloudwatchlogs.DescribeLogGroupsInput{NextToken: nextToken})
		if err != nil {
			return nil, err
		}
		groups = append(groups, output.LogGroups...)
		if output.NextToken != nil {
			nextToken = output.NextToken
		} else {
			break
		}
	}
	return groups, nil
}

func fetchLogStreams(client *cloudwatchlogs.Client, logGroupName string) ([]types.LogStream, error) {
	streams := make([]types.LogStream, 0)
	var nextToken *string

	maxLogSteams := 300

	for {
		output, err := client.DescribeLogStreams(context.TODO(), &cloudwatchlogs.DescribeLogStreamsInput{
			LogGroupName: &logGroupName,
			Limit:        aws.Int32(50),
			OrderBy:      types.OrderByLastEventTime,
			Descending:   aws.Bool(true),
			NextToken:    nextToken,
		})
		if err != nil {
			return nil, err
		}
		streams = append(streams, output.LogStreams...)
		if len(streams) >= maxLogSteams {
			break
		}
		if output.NextToken != nil {
			nextToken = output.NextToken
		} else {
			break
		}
	}
	return streams, nil
}

func fetchLogEvents(client *cloudwatchlogs.Client, logGroupName, logStreamName string) ([]types.OutputLogEvent, error) {
	events := make([]types.OutputLogEvent, 0)
	var nextToken *string

	for {
		output, err := client.GetLogEvents(context.TODO(), &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  &logGroupName,
			LogStreamName: &logStreamName,
			StartFromHead: aws.Bool(true),
			//Limit
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		events = append(events, output.Events...)
		if output.NextForwardToken == nil {
			break
		} else if output.NextForwardToken != output.NextForwardToken {
			nextToken = output.NextForwardToken
		} else {
			break
		}
	}
	return events, nil
}

func main() {
	var log *os.File
	if _, ok := os.LookupEnv("DEBUG"); ok {
		var err error
		log, err = os.OpenFile("messages.log", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			os.Exit(1)
		}
	}
	m := initialModel()
	m.log = log
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

}
