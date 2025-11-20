package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
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
	delegate.SetHeight(1)
	delegate.SetSpacing(0)
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
	rawEvents   []types.OutputLogEvent
	filterInput textinput.Model
	filtering   bool
	filterValue string
}

func NewEventsViewer() *EventsViewer {
	vp := viewport.New(50, 50)
	fi := textinput.New()
	fi.Placeholder = "Filter events..."
	fi.CharLimit = 100
	return &EventsViewer{
		Model:       vp,
		filterInput: fi,
	}
}

func (e *EventsViewer) Update(msg tea.Msg) (Component, tea.Cmd) {
	var cmd tea.Cmd
	if e.filtering {
		e.filterInput, cmd = e.filterInput.Update(msg)
		e.filterValue = e.filterInput.Value()
	} else {
		e.Model, cmd = e.Model.Update(msg)
	}
	return e, cmd
}

func (e *EventsViewer) View() string {
	if e.filtering {
		return e.filterInput.View() + "\n" + e.Model.View()
	}
	return e.Model.View()
}

func (e *EventsViewer) SetSize(width, height int) {
	if e.filtering {
		e.Width, e.Height = width, height-1 // Reserve space for filter input
	} else {
		e.Width, e.Height = width, height
	}
	e.filterInput.Width = width - 10
}

func (e *EventsViewer) SetEvents(events []types.OutputLogEvent) {
	e.rawEvents = events
	e.RefreshContent(false)
}

func (e *EventsViewer) StartFiltering() {
	e.filtering = true
	e.filterInput.Focus()
}

func (e *EventsViewer) StopFiltering() {
	e.filtering = false
	e.filterInput.Blur()
}

func (e *EventsViewer) ClearFilter() {
	e.filterValue = ""
	e.filterInput.SetValue("")
}

func (e *EventsViewer) IsFiltering() bool {
	return e.filtering
}

func (e *EventsViewer) RefreshContent(wrapEnabled bool) {
	content := ""
	for _, event := range e.rawEvents {
		message := *event.Message
		// Apply filter if active
		if e.filterValue != "" && !strings.Contains(strings.ToLower(message), strings.ToLower(e.filterValue)) {
			continue
		}
		if wrapEnabled && e.Width > 0 {
			message = wrapText(message, e.Width)
		}
		content += fmt.Sprintf("%s\n", message)
	}
	e.SetContent(content)
}

// wrapText wraps text to the specified width
func wrapText(text string, width int) string {
	if width <= 0 || len(text) <= width {
		return text
	}

	var lines []string
	for len(text) > width {
		lines = append(lines, text[:width])
		text = text[width:]
	}
	if len(text) > 0 {
		lines = append(lines, text)
	}

	return strings.Join(lines, "\n")
}

