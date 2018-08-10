package raft

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/blend/go-sdk/exception"
	"github.com/blend/go-sdk/logger"
)

var (
	_ RPCServer = &HTTPServer{}
)

// NewHTTPServer returns a new roc server.
func NewHTTPServer() *HTTPServer {
	return &HTTPServer{
		bindAddr: DefaultBindAddr,
		timeout:  DefaultServerTimeout,
	}
}

// HTTPServer is the net/http implementation of the raft rpc server components.
type HTTPServer struct {
	bindAddr      string
	timeout       time.Duration
	requestVote   RequestVoteHandler
	appendEntries AppendEntriesHandler
	log           *logger.Logger
	server        *http.Server
}

// WithLogger sets the logger.
func (s *HTTPServer) WithLogger(log *logger.Logger) *HTTPServer {
	s.log = log
	return s
}

// Logger returns the logger.
func (s *HTTPServer) Logger() *logger.Logger {
	return s.log
}

// WithBindAddr sets the bind address.
func (s *HTTPServer) WithBindAddr(bindAddr string) *HTTPServer {
	s.bindAddr = bindAddr
	return s
}

// BindAddr returns the bind address for the rpc server.
func (s *HTTPServer) BindAddr() string {
	return s.bindAddr
}

// WithTimeout sets the server timeout.
func (s *HTTPServer) WithTimeout(d time.Duration) *HTTPServer {
	s.timeout = d
	return s
}

// Timeout returns the server timeout.
func (s *HTTPServer) Timeout() time.Duration {
	return s.timeout
}

// SetAppendEntriesHandler should register the append entries handler.
func (s *HTTPServer) SetAppendEntriesHandler(handler AppendEntriesHandler) {
	s.appendEntries = handler
}

// AppendEntriesHandler returns the append entries handler.
func (s *HTTPServer) AppendEntriesHandler() AppendEntriesHandler {
	return s.appendEntries
}

// SetRequestVoteHandler should register the request vote handler.
func (s *HTTPServer) SetRequestVoteHandler(handler RequestVoteHandler) {
	s.requestVote = handler
}

// RequestVoteHandler returns the request vote handler.
func (s *HTTPServer) RequestVoteHandler() RequestVoteHandler {
	return s.requestVote
}

func (s *HTTPServer) handle(action http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		// without a logger, just run the action and if panic's happen
		// let them bubble up
		if s.log == nil {
			action(w, req)
			return
		}

		// set up panic handler to log to fatal
		defer func() {
			if r := recover(); r != nil {
				s.log.Fatal(exception.New(r))
			}
		}()
		// trigger handler start
		s.log.Trigger(logger.NewHTTPRequestEvent(req).WithFlag(FlagRPCHandlerStart))

		// set up triggering handler complete
		start := time.Now()
		instrumented := logger.NewResponseWriter(w)
		defer func() {
			s.log.Trigger(logger.NewHTTPResponseEvent(req).WithFlag(FlagRPCHandler).
				WithStatusCode(instrumented.StatusCode()).
				WithElapsed(time.Since(start)).
				WithContentLength(instrumented.ContentLength()))
		}()

		// run the action with an instrumented response writer
		action(instrumented, req)
		return
	}
}

func (s *HTTPServer) appendEntriesHandler(w http.ResponseWriter, req *http.Request) {
	var args AppendEntries
	if err := s.decode(&args, req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var res AppendEntriesResults
	s.appendEntries(&args, &res)
	w.WriteHeader(http.StatusOK)
	if err := s.encode(res, w); err != nil {
		if s.log != nil {
			s.log.Error(err)
		}
	}
}

func (s *HTTPServer) requestVoteHandler(w http.ResponseWriter, req *http.Request) {
	var args RequestVote
	if err := s.decode(&args, req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var res RequestVoteResults
	s.requestVote(&args, &res)
	w.WriteHeader(http.StatusOK)
	if err := s.encode(res, w); err != nil {
		if s.log != nil {
			s.log.Error(err)
		}
	}
}

func (s *HTTPServer) decode(obj interface{}, req *http.Request) error {
	if req.Body == nil {
		return exception.New("request body unset")
	}
	defer req.Body.Close()
	return exception.New(json.NewDecoder(req.Body).Decode(obj))
}

func (s *HTTPServer) encode(obj interface{}, w http.ResponseWriter) error {
	return exception.New(json.NewEncoder(w).Encode(obj))
}

// createServer creates the http server that handles requests.
func (s *HTTPServer) createServer() *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/"+RPCMethodRequestVote, s.handle(s.requestVoteHandler))
	mux.HandleFunc("/"+RPCMethodAppendEntries, s.handle(s.appendEntriesHandler))
	return &http.Server{
		Addr:         s.bindAddr,
		ReadTimeout:  s.timeout,
		WriteTimeout: s.timeout,
		Handler:      mux,
	}
}

// Start starts the server.
func (s *HTTPServer) Start() error {
	if s.log != nil {
		s.log.Infof("rpc server starting, listening on %s", s.bindAddr)
	}
	s.server = s.createServer()
	go s.server.ListenAndServe()
	return nil
}

// Stop stops the server.
// It allows up to a second for the shutdown to process.
func (s *HTTPServer) Stop() error {
	if s.server == nil {
		return nil
	}
	timeout, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()
	return exception.Wrap(s.server.Shutdown(timeout))
}
