package raft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/blend/go-sdk/exception"
	"github.com/blend/go-sdk/logger"
)

var (
	// Assert RPCClient is a client.
	_ RPCClient = &HTTPClient{}
)

// NewRPCClient creates a new rpc client.
func NewRPCClient(remoteAddr string) *HTTPClient {
	return &HTTPClient{
		remoteAddr: remoteAddr,
		transport:  &http.Transport{},
		client:     &http.Client{},
		timeout:    DefaultClientTimeout,
	}
}

// HTTPClient is the net/rpc client to talk to other nodes.
type HTTPClient struct {
	sync.Mutex

	remoteAddr string
	log        *logger.Logger

	transport *http.Transport
	client    *http.Client

	timeout time.Duration
}

// Timeout is the timeout for dialing new connections
func (c *HTTPClient) Timeout() time.Duration {
	return c.timeout
}

// WithTimeout sets the DialTimeout
func (c *HTTPClient) WithTimeout(d time.Duration) *HTTPClient {
	c.timeout = d
	return c
}

// WithLogger sets the logger.
func (c *HTTPClient) WithLogger(log *logger.Logger) *HTTPClient {
	c.log = log
	return c
}

// Logger returns the logger.
func (c *HTTPClient) Logger() *logger.Logger {
	return c.log
}

// WithRemoteAddr sets the remote addr.
func (c *HTTPClient) WithRemoteAddr(addr string) *HTTPClient {
	c.remoteAddr = addr
	return c
}

// RemoteAddr returns the remote address.
func (c *HTTPClient) RemoteAddr() string {
	return c.remoteAddr
}

// Open opens the connection.
func (c *HTTPClient) Open() error {
	c.client = &http.Client{
		Timeout:   c.timeout,
		Transport: c.transport,
	}
	return nil
}

// Close is a nop right now.
func (c *HTTPClient) Close() error {
	return nil
}

// RequestVote implements the request vote handler.
func (c *HTTPClient) RequestVote(args *RequestVote) (*RequestVoteResults, error) {
	var res RequestVoteResults
	err := c.callWithTimeout(RPCMethodRequestVote, args, &res)
	if err != nil {
		return nil, exception.New(err)
	}
	return &res, nil
}

// AppendEntries implements the append entries request handler.
func (c *HTTPClient) AppendEntries(args *AppendEntries) (*AppendEntriesResult, error) {
	var res AppendEntriesResult
	err := c.callWithTimeout(RPCMethodAppendEntries, args, &res)
	if err != nil {
		return nil, exception.New(err)
	}
	return &res, nil
}

// call invokes a method with the default call timeout.
func (c *HTTPClient) callWithTimeout(method string, args interface{}, reply interface{}) error {
	reqURL, err := url.Parse(fmt.Sprintf("http://%s/%s", c.remoteAddr, method))
	if err != nil {
		return exception.Wrap(err)
	}

	body, err := c.encode(args)
	if err != nil {
		return exception.Wrap(err)
	}

	req := &http.Request{
		Method: "POST",
		URL:    reqURL,
		Body:   body,
	}

	if c.log != nil {
		defer func() {
			c.log.Trigger(logger.NewHTTPRequestEvent(req).WithFlag("rpc.call").WithHeadings(c.remoteAddr))
		}()
	}

	res, err := c.client.Do(req)
	if err != nil {
		return exception.Wrap(err)
	}
	if res.StatusCode > 299 {
		return exception.New("non-2xx returned from rpc server").WithMessagef("status code returned: %d", res.StatusCode)
	}

	if err := c.decode(reply, res.Body); err != nil {
		return err
	}
	return nil
}

func (c *HTTPClient) encode(obj interface{}) (io.ReadCloser, error) {
	buffer := new(bytes.Buffer)

	if err := json.NewEncoder(buffer).Encode(obj); err != nil {
		return nil, exception.New(err)
	}
	return ioutil.NopCloser(buffer), nil
}

func (c *HTTPClient) decode(obj interface{}, contents io.ReadCloser) error {
	if contents == nil {
		return exception.New("response body unset; cannot continue")
	}
	defer contents.Close()
	return exception.New(json.NewDecoder(contents).Decode(&obj))
}

func (c *HTTPClient) err(err error) error {
	if c.log != nil && err != nil {
		c.log.Error(err)
	}
	return err
}
