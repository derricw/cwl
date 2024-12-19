package main

import (
	"context"
	"fmt"
	//"log"
	"os"

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
	return logStreamMsg(logStreams)
}

type logGroupMsg []types.LogGroup
type logStreamMsg []types.LogStream

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
	groupsList     list.Model
	streamsList    list.Model
	viewport       viewport.Model
	logGroups      []types.LogGroup
	logStreams     []types.LogStream
	groupSelected  list.Item
	streamSelected list.Item
	mode           mode
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
		for _, stream := range msg {
			items = append(items, item{
				title: *stream.LogStreamName,
			})
		}
		m.streamsList.SetItems(items)
		m.logStreams = msg
	}

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
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if !m.groupsList.SettingFilter() {
				m.groupSelected = m.groupsList.SelectedItem()
				m.mode = Streams
				return m, func() tea.Msg {
					selectedLogGroup := m.logGroups[m.groupsList.Index()]
					return getLogStreams(*selectedLogGroup.LogGroupName)
				}
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.groupsList.SetSize(msg.Width-h, msg.Height-v)
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
				m.streamSelected = m.streamsList.SelectedItem()
				m.mode = Page
				return m, nil
			}
		case "esc":
			m.mode = Groups
			return m, nil
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.streamsList.SetSize(msg.Width-h, msg.Height-v)
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
	m := model{
		groupsList:  list.New(groups, list.NewDefaultDelegate(), 0, 0),
		streamsList: list.New(streams, list.NewDefaultDelegate(), 0, 0),
		viewport:    viewport.New(50, 50),
	}
	m.groupsList.Title = "Log Groups"
	m.streamsList.Title = "Log Streams"
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
	output, err := client.DescribeLogGroups(context.TODO(), &cloudwatchlogs.DescribeLogGroupsInput{})
	if err != nil {
		return nil, err
	}
	return output.LogGroups, nil
}

func fetchLogStreams(client *cloudwatchlogs.Client, logGroupName string) ([]types.LogStream, error) {
	output, err := client.DescribeLogStreams(context.TODO(), &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: &logGroupName,
		Limit:        aws.Int32(50),
		OrderBy:      types.OrderByLastEventTime,
	})
	if err != nil {
		return nil, err
	}
	return output.LogStreams, nil
}

func main() {
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

}
