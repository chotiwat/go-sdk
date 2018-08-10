package raft

// Entry is a raft log entry, it consists of an index and contents.
type Entry struct {
	Index    uint64
	Contents []byte
}
