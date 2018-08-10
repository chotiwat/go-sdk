package raft

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/blend/go-sdk/assert"
)

func TestNewRPCServer(t *testing.T) {
	assert := assert.New(t)

	s := NewRPCServer()
	assert.Nil(s.Logger())
	assert.Equal(DefaultBindAddr, s.BindAddr())
	assert.Equal(DefaultServerTimeout, s.Timeout())
}

func TestRPCServerDecode(t *testing.T) {
	assert := assert.New(t)

	s := NewRPCServer()

	assert.NotNil(s.decode(nil, &http.Request{}))

	output := bytes.NewBuffer([]byte(`{
	"foo":"bar",
	"buzz":"jazz"
}`))

	req := &http.Request{Body: ioutil.NopCloser(output)}
	verify := make(map[string]interface{})
	assert.Nil(s.decode(&verify, req))

	assert.Equal("bar", verify["foo"])
	assert.Equal("jazz", verify["buzz"])
}

func TestRPCServerEncode(t *testing.T) {
	assert := assert.New(t)

	s := NewRPCServer()
	things := map[string]interface{}{
		"foo":  "bar",
		"buzz": "jazz",
	}

	output := new(bytes.Buffer)
	m := NewMockResponseWriter(output)
	assert.Nil(s.encode(things, m))

	verify := make(map[string]interface{})
	assert.Nil(json.Unmarshal(output.Bytes(), &verify))
	assert.Equal("bar", verify["foo"])
	assert.Equal("jazz", verify["buzz"])
}

func TestRPCServerCreateServer(t *testing.T) {
	assert := assert.New(t)

	s := NewRPCServer()
	hs := s.createServer()

	assert.NotNil(hs)
	assert.NotNil(hs.Handler)
	mux, isMux := hs.Handler.(*http.ServeMux)
	assert.True(isMux)
	assert.NotNil(mux)

	assert.Equal(DefaultServerTimeout, hs.ReadTimeout)
	assert.Equal(DefaultServerTimeout, hs.WriteTimeout)
}
