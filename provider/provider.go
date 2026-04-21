// Package provider defines the generic backend interface and domain types
// used by the TUI. Backends (CloudWatch, MLflow, etc.) implement the Backend
// interface to provide log data from different sources.
package provider

import "time"

type LogGroup struct {
	Name string
	Desc string // ARN for CloudWatch, experiment ID for MLflow, etc.
}

type LogStream struct {
	Name          string
	LastEventTime *time.Time
}

type LogEvent struct {
	Message   string
	Timestamp *time.Time
}

// Backend abstracts log fetching so the TUI works with any log source.
type Backend interface {
	FetchGroups(pattern string) ([]LogGroup, error)
	FetchStreamsStreaming(group string, callback func([]LogStream) error) error
	FetchEventsStreaming(group, stream string, callback func([]LogEvent) error) error
	FetchLastEvents(group, stream string, limit int) ([]LogEvent, error)
	FetchNewEvents(group, stream string, since *time.Time) ([]LogEvent, error)
}
