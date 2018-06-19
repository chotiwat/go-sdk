package logger

import (
	"time"
)

// these are compile time assertions
var (
	_ Event            = &MessageEvent{}
	_ EventHeadings    = &MessageEvent{}
	_ EventLabels      = &MessageEvent{}
	_ EventAnnotations = &MessageEvent{}
)

// NewFrontendEvent creates a new query event.
func NewFrontendEvent(body []byte, timestamp time.Time) *FrontendEvent {
	return &FrontendEvent{
		flag:      Frontend,
		timestamp: timestamp,
		body:      body,
	}
}

// FrontendEvent represents an event on the frontend
type FrontendEvent struct {
	*EventMeta

	flag      Flag
	timestamp time.Time
	body      []byte
}

// Flag returns the event flag
func (f *FrontendEvent) Flag() Flag {
	return f.flag
}

// Timestamp returns the event timestamp
func (f *FrontendEvent) Timestamp() time.Time {
	return f.timestamp
}

// String returns the event as a string
func (f *FrontendEvent) String() string {
	return string(f.body)
}

// WithLabel sets a label on the event for later filtering.
func (f *FrontendEvent) WithLabel(key, value string) *FrontendEvent {
	f.AddLabelValue(key, value)
	return f
}

// Labels returns the event's labels
func (f *FrontendEvent) Labels() map[string]string {
	return f.labels
}

// Annotations is a no op
func (f *FrontendEvent) Annotations() map[string]string {
	return make(map[string]string)
}
