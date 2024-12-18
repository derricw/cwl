package main

import (
	"context"
	"fmt"
	//"log"
	"os"

	//"github.com/aws/aws-sdk-go-v2/aws"
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

type logGroupMsg []types.LogGroup

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
	list          list.Model
	viewport      viewport.Model
	logGroups     []types.LogGroup
	groupSelected list.Item
	mode          mode
}

func (m model) Init() tea.Cmd {
	return getLogGroups
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		switch k {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if !m.list.SettingFilter() {
				m.groupSelected = m.list.SelectedItem()
				m.mode = Page
				return m, nil
			}
		case "esc":
			if m.mode == Page {
				m.groupSelected = nil
				m.mode = Groups
				return m, nil
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case logGroupMsg:
		items := []list.Item{}
		for _, group := range msg {
			items = append(items, item{
				title: *group.LogGroupName,
				desc:  *group.LogGroupArn,
			})
		}
		m.list.SetItems(items)
		m.logGroups = msg
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.mode == Groups {
		return docStyle.Render(m.list.View())
	}
	return docStyle.Render(m.viewport.View())
}

func initialModel() model {
	items := []list.Item{}
	m := model{list: list.New(items, list.NewDefaultDelegate(), 0, 0)}
	m.list.Title = "Log Groups"
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

func main() {
	m := initialModel()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

}
