package main

import (
	"bytes"
	"fmt"

	"github.com/blend/go-sdk/logger"
)

// NewCustomEvent returns a new custom event.
// It is helpful to use a constructor syou can initialize the event meta.
func NewCustomEvent(userID, sessionID, context string) CustomEvent {
	return CustomEvent{
		EventMeta: logger.NewEventMeta(logger.Flag("custom_event")),
		UserID:    userID,
		SessionID: sessionID,
		Context:   context,
	}
}

// CustomEvent is a custom logger event.
type CustomEvent struct {
	*logger.EventMeta // this embeds a bunch of common fields into our event.

	UserID    string // something domain specific
	SessionID string // something domain specific
	Context   string // something domain specific
}

// WriteText implements logger.TextWritable.
// It is optional, but very much encouraged.
// It takes a formatter and a buffer reference that you push data into.
// This lets the logger re-use buffers.
func (ce CustomEvent) WriteText(tf logger.TextFormatter, buf *bytes.Buffer) {
	buf.WriteString(ce.UserID)
	buf.WriteRune(logger.RuneSpace)
	buf.WriteString(ce.SessionID)
	buf.WriteRune(logger.RuneSpace)
	buf.WriteString(ce.Context)
}

// WriteJSON implements logger.JSONWritable.
// It is a function that returns just the custom fields on our object as a map,
// to be serialized with the rest of the fields.
func (ce CustomEvent) WriteJSON() logger.JSONObj {
	return logger.JSONObj{
		"userID":    ce.UserID,
		"sessionID": ce.SessionID,
		"context":   ce.Context,
	}
}

// NewCustomEventListener returns a type shim for the logger.
func NewCustomEventListener(listener func(ce CustomEvent)) logger.Listener {
	return func(e logger.Event) {
		listener(e.(CustomEvent))
	}
}

func main() {

	// make a text logger.
	text := logger.NewText().WithFlags(logger.AllFlags())
	// make a json logger
	js := logger.NewJSON().WithFlags(logger.AllFlags())

	event := NewCustomEvent("bailey", "session0", "Console Demo")

	text.SyncTrigger(event)
	js.SyncTrigger(event)

	listener := logger.New().WithFlags(logger.AllFlags())
	listener.Listen(logger.Flag("custom_event"), "demo", NewCustomEventListener(func(ce CustomEvent) {
		fmt.Println("listener got event")
	}))

	listener.SyncTrigger(event)
}
