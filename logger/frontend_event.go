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
func NewFrontendEvent(body []byte) *FrontendEvent {
	return &FrontendEvent{
		EventMeta: NewEventMeta(FEEvent),
		body:      body,
	}
}

// FrontendEvent represents an event on the frontend
type FrontendEvent struct {
	*EventMeta

	body []byte
}

// Body returns the event body.
func (f *FrontendEvent) Body() []byte {
	return f.body
}

// Flag returns the event flag
func (f *FrontendEvent) Flag() Flag {
	return f.flag
}

// Timestamp returns the event timestamp
func (f *FrontendEvent) Timestamp() time.Time {
	return f.ts
}

// String returns the event as a string
func (f *FrontendEvent) String() string {
	return string(f.body)
}

// WithBody sets the body.
func (f *FrontendEvent) WithBody(body []byte) *FrontendEvent {
	f.body = body
	return f
}

// WithTimestamp sets the timestamp.
func (f *FrontendEvent) WithTimestamp(ts time.Time) *FrontendEvent {
	f.ts = ts
	return f
}

// WithLabel sets a label on the event for later filtering.
func (f *FrontendEvent) WithLabel(key, value string) *FrontendEvent {
	f.AddLabelValue(key, value)
	return f
}

// WithAnnotation adds an annotation to the event.
func (f *FrontendEvent) WithAnnotation(key, value string) *FrontendEvent {
	f.AddAnnotationValue(key, value)
	return f
}

// WithFlag sets the flag.
func (f *FrontendEvent) WithFlag(flag Flag) *FrontendEvent {
	f.flag = flag
	return f
}

// Labels returns the event's labels
func (f *FrontendEvent) Labels() map[string]string {
	return f.labels
}

// Annotations returns the event's annotations
func (f *FrontendEvent) Annotations() map[string]string {
	return f.annotations
}
