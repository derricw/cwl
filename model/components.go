package model

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Component interface for all UI components
type Component interface {
	Update(msg tea.Msg) (Component, tea.Cmd)
	View() string
	SetSize(width, height int)
}

// GroupsList component
type GroupsList struct {
	list.Model
}

func NewGroupsList() *GroupsList {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Log Groups"
	return &GroupsList{Model: l}
}

func (g *GroupsList) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd
	g.Model, cmd = g.Model.Update(msg)
	return g, cmd
}

func (g *GroupsList) View() string {
	return g.Model.View()
}

func (g *GroupsList) SetSize(width, height int) {
	g.Model.SetSize(width, height)
}

func (g *GroupsList) SetGroups(groups []types.LogGroup) {
	items := make([]list.Item, 0, len(groups))
	for _, group := range groups {
		items = append(items, item{
			title: *group.LogGroupName,
			desc:  *group.LogGroupArn,
		})
	}
	g.SetItems(items)
}

// StreamsList component
type StreamsList struct {
	list.Model
	groupName string
}

func NewStreamsList() *StreamsList {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	l := list.New([]list.Item{}, delegate, 0, 0)
	return &StreamsList{Model: l}
}

func (s *StreamsList) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd
	s.Model, cmd = s.Model.Update(msg)
	return s, cmd
}

func (s *StreamsList) View() string {
	return s.Model.View()
}

func (s *StreamsList) SetSize(width, height int) {
	s.Model.SetSize(width, height)
}

func (s *StreamsList) SetStreams(groupName string, streams []types.LogStream) {
	s.groupName = groupName
	s.Title = fmt.Sprintf("Log Streams: %s", groupName)
	
	items := make([]list.Item, 0, len(streams))
	for _, stream := range streams {
		items = append(items, item{
			title: *stream.LogStreamName,
			desc:  time.Unix(0, *stream.LastEventTimestamp*1000000).Format("2006-01-02 15:04:05"),
		})
	}
	s.SetItems(items)
}

// EventsViewer component
type EventsViewer struct {
	viewport.Model
}

func NewEventsViewer() *EventsViewer {
	vp := viewport.New(50, 50)
	return &EventsViewer{Model: vp}
}

func (e *EventsViewer) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd
	e.Model, cmd = e.Model.Update(msg)
	return e, cmd
}

func (e *EventsViewer) View() string {
	return e.Model.View()
}

func (e *EventsViewer) SetSize(width, height int) {
	e.Width, e.Height = width, height
}

func (e *EventsViewer) SetEvents(events []types.OutputLogEvent) {
	content := ""
	for _, event := range events {
		content += fmt.Sprintf("%s\n", *event.Message)
	}
	e.SetContent(content)
}