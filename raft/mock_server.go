package raft

var (
	// assert mock server is a server.
	_ RPCServer = &MockServer{}
)

// NewMockServer is a mocked server.
func NewMockServer() *MockServer {
	return &MockServer{}
}

// MockServer is a mocked server.
type MockServer struct {
	appendEntriesHandler AppendEntriesHandler
	requestVoteHandler   RequestVoteHandler
}

// Start is a no op.
func (ms *MockServer) Start() error { return nil }

// Stop is a no op.
func (ms *MockServer) Stop() error { return nil }

// SetAppendEntriesHandler should register the append entries handler.
func (ms *MockServer) SetAppendEntriesHandler(handler AppendEntriesHandler) {
	ms.appendEntriesHandler = handler
}

// AppendEntriesHandler returns the append entries handler.
func (ms *MockServer) AppendEntriesHandler() AppendEntriesHandler { return ms.appendEntriesHandler }

// SetRequestVoteHandler should register the request vote handler.
func (ms *MockServer) SetRequestVoteHandler(handler RequestVoteHandler) {
	ms.requestVoteHandler = handler
}

// RequestVoteHandler returns the request vote handler.
func (ms *MockServer) RequestVoteHandler() RequestVoteHandler { return ms.requestVoteHandler }
