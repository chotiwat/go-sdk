package raft

// AppendEntries is a log propagation request.
type AppendEntries struct {
	// ID is the node id sending the append entries rpc.
	ID string
	// Term represents the leader's term
	Term uint64
	// PrevLogIndex is the index of the log entry immediately preceding new ones
	PrevLogIndex uint64
	// PrevLogterm is the term of the PrevLogIndex entry
	PrevLogTerm uint64
	// LeaderCommit is the leader's Commit Index
	LeaderCommit uint64
	// Entries are the log entries to send.
	// They can be formatted however you'd like.
	Entries []Entry
}

// AppendEntriesResults is the response from an append entries request.
type AppendEntriesResults struct {
	// ID is the node id that responded.
	ID string
	// Term is the current term, for leader to update itself
	Term uint64
	// Success is true if follower contained entry matching the PrevLogIndex and PrevLogTerm
	Success bool
}
