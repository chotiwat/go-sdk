package logger

import (
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
)

func TestFrontendEventInterfaces(t *testing.T) {
	assert := assert.New(t)

	fe := NewFrontendEvent([]byte("this is a test")).
		WithLabel("foo", "bar")

	eventProvider, isEvent := MarshalEvent(fe)
	assert.True(isEvent)
	assert.Equal(FEEvent, eventProvider.Flag())
	assert.False(eventProvider.Timestamp().IsZero())

	metaProvider, isMetaProvider := MarshalEventMetaProvider(fe)
	assert.True(isMetaProvider)
	assert.Equal("bar", metaProvider.Labels()["foo"])
}

func TestFrontendEventProperties(t *testing.T) {
	assert := assert.New(t)

	f := NewFrontendEvent([]byte(""))
	assert.False(f.Timestamp().IsZero())
	assert.True(f.WithTimestamp(time.Time{}).Timestamp().IsZero())

	assert.Empty(f.Labels())
	assert.Equal("bar", f.WithLabel("foo", "bar").Labels()["foo"])

	assert.Empty(f.Annotations())
	assert.Equal("zar", f.WithAnnotation("moo", "zar").Annotations()["moo"])

	assert.Equal(FEEvent, f.Flag())
	assert.Equal(Error, f.WithFlag(Error).Flag())

	assert.Empty(f.Body())
	assert.Equal([]byte("Body"), f.WithBody([]byte("Body")).Body())

	assert.Empty(f.Headings())
}
