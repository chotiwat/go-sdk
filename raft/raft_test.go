package raft

import (
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
	"github.com/blend/go-sdk/logger"
	"github.com/blend/go-sdk/uuid"
)

func TestNew(t *testing.T) {
	assert := assert.New(t)

	r := New()
	assert.NotEmpty(r.ID())
	assert.Equal(Follower, r.State())
	assert.Equal(DefaultElectionTimeout, r.ElectionTimeout())
	assert.Equal(DefaultLeaderCheckInterval, r.LeaderCheckInterval())
	assert.Equal(DefaultHeartbeatInterval, r.HeartbeatInterval())
}

func TestNewFromCfg(t *testing.T) {
	assert := assert.New(t)

	r := NewFromConfig(&Config{
		ID:                  "test-node",
		HeartbeatInterval:   5 * time.Second,
		LeaderCheckInterval: 6 * time.Second,
		ElectionTimeout:     7 * time.Second,
	})
	assert.Equal("test-node", r.ID())
	assert.Equal(Follower, r.State())
	assert.Equal(5*time.Second, r.HeartbeatInterval())
	assert.Equal(6*time.Second, r.LeaderCheckInterval())
	assert.Equal(7*time.Second, r.ElectionTimeout())
}

func TestRaftProperties(t *testing.T) {
	assert := assert.New(t)

	r := New()

	r.WithID("test-node")
	assert.Equal("test-node", r.ID())

	r.state = Candidate
	assert.Equal(Candidate, r.State())

	r.votedFor = "not-test-node"
	assert.Equal("not-test-node", r.VotedFor())

	r.currentTerm = 123
	assert.Equal(123, r.CurrentTerm())

	r.lastLeaderContact = time.Date(1999, 01, 01, 0, 0, 0, 0, time.UTC)
	assert.Equal(1999, r.LastLeaderContact().Year())

	assert.Nil(r.LeaderHandler())
	r.SetLeaderHandler(func() {})
	assert.NotNil(r.LeaderHandler())

	assert.Nil(r.CandidateHandler())
	r.SetCandidateHandler(func() {})
	assert.NotNil(r.CandidateHandler())

	assert.Nil(r.FollowerHandler())
	r.SetFollowerHandler(func() {})
	assert.NotNil(r.FollowerHandler())

	assert.Nil(r.Logger())
	r.WithLogger(logger.None())
	assert.NotNil(r.Logger())

	assert.Empty(r.Peers())
	r.WithPeer(NoOpTransport("hello"))
	assert.NotEmpty(r.Peers())
	r.WithPeers()
	assert.Empty(r.Peers())

	assert.Nil(r.Server())
	r.WithServer(NewMockServer())
	assert.NotNil(r.Server())

	r.WithElectionTimeout(5 * time.Second)
	assert.Equal(5*time.Second, r.ElectionTimeout())

	r.WithLeaderCheckInterval(6 * time.Second)
	assert.Equal(6*time.Second, r.LeaderCheckInterval())

	r.WithHeartbeatInterval(7 * time.Second)
	assert.Equal(7*time.Second, r.HeartbeatInterval())
}

func TestRaftCountVotes(t *testing.T) {
	assert := assert.New(t)

	r := New()

	assert.Equal(-1, r.voteOutcome(0, 0))
	assert.Equal(-1, r.voteOutcome(1, 0))

	assert.Equal(-1, r.voteOutcome(0, 1))
	assert.Equal(-1, r.voteOutcome(1, 1))

	assert.Equal(-1, r.voteOutcome(0, 2))
	assert.Equal(0, r.voteOutcome(1, 2))
	assert.Equal(1, r.voteOutcome(2, 2))

	assert.Equal(-1, r.voteOutcome(0, 3))
	assert.Equal(-1, r.voteOutcome(1, 3))
	assert.Equal(1, r.voteOutcome(2, 3))
	assert.Equal(1, r.voteOutcome(3, 3))

	assert.Equal(-1, r.voteOutcome(0, 4))
	assert.Equal(-1, r.voteOutcome(1, 4))
	assert.Equal(0, r.voteOutcome(2, 4))
	assert.Equal(1, r.voteOutcome(3, 4))
	assert.Equal(1, r.voteOutcome(4, 4))

	assert.Equal(-1, r.voteOutcome(0, 4))
	assert.Equal(-1, r.voteOutcome(1, 4))
	assert.Equal(0, r.voteOutcome(2, 4))
	assert.Equal(1, r.voteOutcome(3, 4))
	assert.Equal(1, r.voteOutcome(4, 4))
}

func TestRaftSoloStart(t *testing.T) {
	assert := assert.New(t)

	solo := New()
	solo.Start()
	assert.Empty(solo.Peers())
	assert.Nil(solo.Server())
	assert.Equal(Leader, solo.State())
	assert.Nil(solo.leaderCheckTicker)
	assert.Nil(solo.heartbeatTicker)
	assert.True(solo.IsRunning())
}

func TestRaftStart(t *testing.T) {
	assert := assert.New(t)

	node := New()
	node.WithServer(NewMockServer())
	node.WithPeer(NoOpTransport("one"))
	node.WithPeer(NoOpTransport("two"))
	node.WithPeer(NoOpTransport("three"))

	node.Start()
	defer node.Stop()

	assert.NotNil(node.Server())
	assert.Len(node.Peers(), 3)
	assert.True(node.IsRunning())

	assert.True(node.leaderCheckTicker.IsRunning())
	assert.True(node.heartbeatTicker.IsRunning())
}

func TestRaftStop(t *testing.T) {
	assert := assert.New(t)

	node := New()
	node.WithServer(NewMockServer())
	node.WithPeer(NoOpTransport("one"))
	node.WithPeer(NoOpTransport("two"))
	node.WithPeer(NoOpTransport("three"))

	node.Start()
	<-node.latch.NotifyStarted()

	assert.NotNil(node.Server())
	assert.Len(node.Peers(), 3)

	assert.True(node.leaderCheckTicker.IsRunning())
	assert.True(node.heartbeatTicker.IsRunning())

	node.Stop()

	assert.Nil(node.leaderCheckTicker)
	assert.Nil(node.heartbeatTicker)
}

func TestRaftAppendEntriesHandler(t *testing.T) {
	assert := assert.New(t)

	r := New()
	r.state = Leader

	var res AppendEntriesResult
	assert.Nil(r.receiveEntries(&AppendEntries{
		ID:   "test-node",
		Term: 1,
	}, &res))

	assert.True(res.Success)
	assert.Equal(r.ID(), res.ID)
	assert.Equal(1, res.Term)

	assert.Equal(Follower, r.State(), "append entries should set the node as a follower")
	assert.Equal(1, r.CurrentTerm())
	assert.False(r.LastLeaderContact().IsZero())
	assert.True(r.lastVoteGranted.IsZero())
	assert.Empty(r.votedFor)
}

func TestRaftAppendEntriesHandlerInvalidTerm(t *testing.T) {
	assert := assert.New(t)

	r := New()
	r.state = Candidate
	r.currentTerm = 2

	var res AppendEntriesResult
	assert.Nil(r.receiveEntries(&AppendEntries{
		ID:   "test-node",
		Term: 1,
	}, &res))

	assert.False(res.Success)
	assert.Equal(r.ID(), res.ID)
	assert.Equal(2, res.Term)

	assert.Equal(Candidate, r.State())
	assert.Equal(2, r.CurrentTerm())
	assert.True(r.LastLeaderContact().IsZero())
	assert.True(r.lastVoteGranted.IsZero())
	assert.Empty(r.votedFor)
}

func TestRaftRequestVoteHandler(t *testing.T) {
	assert := assert.New(t)

	r := New()
	r.state = Candidate

	var res RequestVoteResults

	assert.Nil(r.vote(&RequestVote{
		ID:   "test-node",
		Term: 1,
	}, &res))

	assert.Equal(r.ID(), res.ID)
	assert.Equal(1, res.Term)
	assert.True(res.Granted)

	assert.Equal(Follower, r.State())
	assert.Equal(1, r.CurrentTerm())
	assert.Equal("test-node", r.VotedFor())
	assert.False(r.lastVoteGranted.IsZero())
}

func TestRaftRequestVoteHandlerAlreadyVoted(t *testing.T) {
	assert := assert.New(t)

	r := New()
	r.state = Follower
	r.currentTerm = 1
	r.lastVoteGranted = r.now()
	r.votedFor = "test-node-2"

	var res RequestVoteResults
	assert.Nil(r.vote(&RequestVote{
		ID:   "test-node",
		Term: 1,
	}, &res))

	assert.Equal("test-node-2", res.ID)
	assert.Equal(1, res.Term)
	assert.False(res.Granted)

	assert.Equal(Follower, r.State())
	assert.Equal(1, r.CurrentTerm())
	assert.Equal("test-node-2", r.VotedFor())
	assert.False(r.lastVoteGranted.IsZero())
}

func TestRaftProcessRequestVoteResults(t *testing.T) {
	assert := assert.New(t)

	r := New().WithID("one").WithPeers(NoOpTransport("two"), NoOpTransport("three"))

	results := make(chan *RequestVoteResults, 2)
	results <- &RequestVoteResults{Granted: true}
	results <- &RequestVoteResults{Granted: true}

	assert.Equal(ElectionVictory, r.processRequestVoteResults(results))

	results = make(chan *RequestVoteResults, 2)
	results <- &RequestVoteResults{Granted: false}
	results <- &RequestVoteResults{Granted: false}

	assert.Equal(ElectionLoss, r.processRequestVoteResults(results))
}

func TestRaftProcessAppendEntriesResult(t *testing.T) {
	assert := assert.New(t)

	r := New().WithID("one").WithPeers(NoOpTransport("two"), NoOpTransport("three"))

	results := make(chan *AppendEntriesResult, 2)
	results <- &AppendEntriesResult{Success: true}
	results <- &AppendEntriesResult{Success: true}

	assert.Equal(ElectionVictory, r.processAppendEntriesResult(results))

	results = make(chan *AppendEntriesResult, 2)
	results <- &AppendEntriesResult{Success: false}
	results <- &AppendEntriesResult{Success: false}

	assert.Equal(ElectionLoss, r.processAppendEntriesResult(results))
}

func TestRaftTransitionTo(t *testing.T) {
	assert := assert.New(t)

	r := New()
	r.state = Follower

	didCallHandler := make(chan struct{})
	r.followerHandler = func() {
		close(didCallHandler)
	}

	go r.transition(Follower)
	r.state = Leader

	go r.transition(Follower)
	<-didCallHandler
	assert.Equal(Follower, r.state)

	r.state = Leader
	didCallHandler = make(chan struct{})
	r.leaderHandler = func() {
		close(didCallHandler)
	}

	go r.transition(Leader)
	r.state = Candidate

	go r.transition(Leader)
	<-didCallHandler
	assert.Equal(Leader, r.state)

	r.state = Candidate
	didCallHandler = make(chan struct{})
	r.candidateHandler = func() {
		close(didCallHandler)
	}

	go r.transition(Candidate)

	r.state = Follower
	go r.transition(Candidate)
	<-didCallHandler
	assert.Equal(Candidate, r.state)
}

func TestRaftAppendEntriesDemotesSelf(t *testing.T) {
	assert := assert.New(t)

	r := New()
	r.state = Leader
	calledHandler := make(chan struct{})
	r.followerHandler = func() {
		close(calledHandler)
	}
	assert.True(r.IsState(Leader))

	var response AppendEntriesResult
	go r.receiveEntries(&AppendEntries{
		ID:   uuid.V4().String(),
		Term: r.currentTerm + 1,
	}, &response)
	<-calledHandler
	assert.True(r.IsState(Follower))
}
