package raft

import (
	"fmt"
	"sync"
	"time"

	"github.com/blend/go-sdk/async"
	"github.com/blend/go-sdk/logger"
	"github.com/blend/go-sdk/uuid"
)

const (
	// ErrAlreadyStarted is returned if you call start on a started node.
	ErrAlreadyStarted = Error("raft is already started")
	// ErrNotRunning is returned if you try and call stop on a stopped node.
	ErrNotRunning = Error("raft is not running")
	// ErrServerUnset is returned if you try and start a node w/o configuring the server.
	ErrServerUnset = Error("raft rpc server unset")
)

// New creates a new empty raft node.
func New() *Raft {
	return &Raft{
		id:                  uuid.V4().String(),
		state:               Follower,
		latch:               &async.Latch{},
		electionTimeout:     DefaultElectionTimeout,
		leaderCheckInterval: DefaultLeaderCheckInterval,
		heartbeatInterval:   DefaultHeartbeatInterval,
	}
}

// NewFromConfig creates a new raft node from a config.
func NewFromConfig(cfg *Config) *Raft {
	return New().
		WithID(cfg.GetID()).
		WithHeartbeatInterval(cfg.GetHeartbeatInterval()).
		WithLeaderCheckInterval(cfg.GetLeaderCheckInterval()).
		WithElectionTimeout(cfg.GetElectionTimeout())
}

// Raft represents a raft node and all the state machine
// componentry required.
type Raft struct {
	sync.Mutex

	id  string
	log *logger.Logger

	electionTimeout time.Duration

	leaderCheckInterval time.Duration
	heartbeatInterval   time.Duration

	// raft state fields
	entriesLog  []Entry
	commitIndex uint64
	lastApplied uint64

	// for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	nextIndex map[string]uint64
	// for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)
	matchIndex map[string]uint64

	// currentTerm is the current election term. starts at zero.
	// incremented every election() and by append entries from new leaders.
	currentTerm       uint64
	votedFor          string
	lastLeaderContact time.Time
	lastVoteGranted   time.Time

	// state is the current fsm state
	state        State
	backoffIndex int32
	stateChanged chan struct{}

	server RPCServer
	peers  []RPCClient

	latch             *async.Latch
	leaderCheckTicker *async.Interval
	heartbeatTicker   *async.Interval

	leaderHandler    func()
	candidateHandler func()
	followerHandler  func()
}

// --------------------------------------------------------------------------------
// properties
// --------------------------------------------------------------------------------

// WithID sets the identifier for the node.
func (r *Raft) WithID(id string) *Raft {
	r.id = id
	return r
}

// ID is the raft node identifier.
func (r *Raft) ID() string {
	return r.id
}

// State returns the current raft state. It is read only.
func (r *Raft) State() State {
	return r.state
}

// IsState returns if the node is a given state.
func (r *Raft) IsState(state State) (output bool) {
	r.Lock()
	output = state == r.state
	r.Unlock()
	return
}

// IsNotState returns if the node is not a given state.
func (r *Raft) IsNotState(state State) (output bool) {
	r.Lock()
	output = state != r.state
	r.Unlock()
	return
}

// IsLeader returns if the node is the leader.
func (r *Raft) IsLeader() (output bool) {
	r.Lock()
	output = Leader == r.state
	r.Unlock()
	return
}

// Latch returns the latch coordinator.
func (r *Raft) Latch() *async.Latch {
	return r.latch
}

// IsRunning returns if the raft node is started.
func (r *Raft) IsRunning() bool {
	return r.latch.IsRunning()
}

// VotedFor returns the current known leader. It is read only.
func (r *Raft) VotedFor() string {
	return r.votedFor
}

// CurrentTerm returns the current raft term. It is read only.
func (r *Raft) CurrentTerm() uint64 {
	return r.currentTerm
}

// LastLeaderContact is the last time we heard from the leader. It is read only.
func (r *Raft) LastLeaderContact() time.Time {
	return r.lastLeaderContact
}

// SetLeaderHandler sets the leader handler.
func (r *Raft) SetLeaderHandler(handler func()) {
	r.leaderHandler = handler
}

// LeaderHandler returns the leader handler.
func (r *Raft) LeaderHandler() func() {
	return r.leaderHandler
}

// SetCandidateHandler sets the leader handler.
func (r *Raft) SetCandidateHandler(handler func()) {
	r.candidateHandler = handler
}

// CandidateHandler returns the candidate handler.
func (r *Raft) CandidateHandler() func() {
	return r.candidateHandler
}

// SetFollowerHandler sets the leader handler.
func (r *Raft) SetFollowerHandler(handler func()) {
	r.followerHandler = handler
}

// FollowerHandler returns the follower handler.
func (r *Raft) FollowerHandler() func() {
	return r.followerHandler
}

// WithLogger sets the logger.
func (r *Raft) WithLogger(log *logger.Logger) *Raft {
	r.log = log
	return r
}

// Logger returns the logger.
func (r *Raft) Logger() *logger.Logger {
	return r.log
}

// WithPeer adds a peer.
func (r *Raft) WithPeer(peer RPCClient) *Raft {
	r.peers = append(r.peers, peer)
	return r
}

// WithPeers sets the peer list.
func (r *Raft) WithPeers(peers ...RPCClient) *Raft {
	r.peers = peers
	return r
}

// Peers returns the raft peers.
func (r *Raft) Peers() []RPCClient {
	return r.peers
}

// WithServer sets the rpc server.
func (r *Raft) WithServer(server RPCServer) *Raft {
	r.server = server
	return r
}

// Server returns the rpc server. func (r *Raft) Server() Server { return r.server } // WithElectionTimeout sets the election timeout. func (r *Raft) WithElectionTimeout(d time.Duration) *Raft { r.electionTimeout = d return r } // ElectionTimeout returns the election timeout. func (r *Raft) ElectionTimeout() time.Duration { return r.electionTimeout } // WithLeaderCheckInterval sets the leader check tick. func (r *Raft) WithLeaderCheckInterval(d time.Duration) *Raft { r.leaderCheckInterval = d return r } // LeaderCheckInterval returns the leader check tick time. func (r *Raft) LeaderCheckInterval() time.Duration { return r.leaderCheckInterval } // WithHeartbeatInterval sets the heartbeat tick. func (r *Raft) WithHeartbeatInterval(d time.Duration) *Raft { r.heartbeatInterval = d return r } // HeartbeatInterval returns the heartbeat tick rate. func (r *Raft) HeartbeatInterval() time.Duration { return r.heartbeatInterval } // -------------------------------------------------------------------------------- // lifecycle // -------------------------------------------------------------------------------- // Start starts the raft node. // It starts internal tickers and starts the rpc server. // It will return an error if the raft node is already started. func (r *Raft) Start() error { r.Lock() defer r.Unlock() if r.latch.IsStarting() || r.latch.IsRunning() { return exception.New(ErrAlreadyStarted) } r.infof("node starting") r.latch.Starting() if len(r.peers) == 0 { r.infof("node operating in solo node configuration") r.transition(Leader) r.latch.Started() r.infof("node started") return nil } if r.server == nil { return exception.New(ErrServerUnset) } // wire up the rpc server. r.server.SetAppendEntriesHandler(r.receiveEntries) r.server.SetRequestVoteHandler(r.vote) if err := r.server.Start(); err != nil { return err } r.leaderCheckTicker = async.NewInterval(r.leaderCheck, r.leaderCheckInterval) r.leaderCheckTicker.Start() r.heartbeatTicker = async.NewInterval(r.heartbeat, r.heartbeatInterval) r.heartbeatTicker.Start() r.latch.Started() r.infof("node started") return nil } // Stop stops the node. // It stops internal tickers, and shuts down the rpc server. func (r *Raft) Stop() error { r.Lock() defer r.Unlock() if !r.latch.IsRunning() { return exception.New(ErrNotRunning) } r.latch.Stopping() if r.leaderCheckTicker != nil { r.leaderCheckTicker.Stop() r.leaderCheckTicker = nil } if r.heartbeatTicker != nil { r.heartbeatTicker.Stop() r.heartbeatTicker = nil } if r.server != nil { return r.server.Stop() } r.latch.Stopped() return nil } // -------------------------------------------------------------------------------- // interval handlers // -------------------------------------------------------------------------------- // LeaderCheck is the action that fires on an interval to check if the leader lease has expired. // If it fails, it triggers an election. func (r *Raft) leaderCheck() error { if r.IsState(Follower) { // if we've never elected a leader, or if the current leader hasn't sent a heartbeat in a while ... // and if we haven't voted yet if r.shouldTriggerElection() { // trigger an election r.err(r.election()) } } return nil } // Heartbeat is the action triggered upon send heartbeat. // This method is fully interlocked. // This method launches a goroutine. func (r *Raft) heartbeat() error { if r.IsNotState(Leader) { return nil } r.sendHeartbeats() return nil } // -------------------------------------------------------------------------------- // rpc handlers // -------------------------------------------------------------------------------- // ReceiveEntries is the rpc server handler for AppendEntries rpc requests. // This method is fully interlocked. func (r *Raft) receiveEntries(args *AppendEntries, res *AppendEntriesResults) error { r.Lock() defer r.Unlock() if args.Term < r.currentTerm { r.debugf("received out of date leader heartbeat (%d vs. %d)", args.Term, r.currentTerm) *res = AppendEntriesResults{ ID:      r.id, Success: false, Term:    r.currentTerm, } return nil } if r.state == Leader { r.debugf("received entries from %s as leader", args.ID) } else if r.state == Candidate { r.debugf("received entries from %s as candidate", args.ID) } r.setFollower() r.currentTerm = args.Term r.lastLeaderContact = r.now() *res = AppendEntriesResults{ ID:      r.id, Success: true, Term:    r.currentTerm, } return nil } // Vote is the rpc server handler for RequestVote rpc requests. // This method is fully interlocked. // It is called when a peer is calling for an election, and the result determines this node's vote. func (r *Raft) vote(args *RequestVote, res *RequestVoteResults) error { r.Lock() defer r.Unlock() // if the term is very out of date if args.Term < r.currentTerm { r.debugf("rejecting request vote from %s, term: %d  (past term)", args.ID, args.Term) *res = RequestVoteResults{ ID:      r.id, Term:    r.currentTerm, Granted: false, } return nil } // if we've voted (this term) and isVoteValid := !r.lastVoteGranted.IsZero() && r.now().Sub(r.lastVoteGranted) < r.electionTimeout votedForCandidate := r.votedFor == args.ID if isVoteValid && !votedForCandidate { r.debugf("rejecting request vote from %s, term: %d (already voted)", args.ID, args.Term) *res = RequestVoteResults{ ID:      r.votedFor, Term:    r.currentTerm, Granted: false, } return nil } r.debugf("accepting request vote from %s, term: %d", args.ID, args.Term) r.transition(Follower) r.votedFor = args.ID r.currentTerm = args.Term r.lastVoteGranted = r.now() *res = RequestVoteResults{ ID:      r.id, Term:    args.Term, Granted: true, } return nil }
// --------------------------------------------------------------------------------
// helper methods
// --------------------------------------------------------------------------------

// Election requests votes from all peers, totalling the results and potentially promoting self to leader.
// It is time bound on the ElectionTimeout.
// It does not interlock during the election as the election can last a while, but does acquire locks for
// some sub calls.
func (r *Raft) election() error {
	r.debugf("election triggered")
	r.resetVoteSafe()
	r.setCandidateSafe()

	if r.shouldStopElection() {
		r.debugf("should stop election; no longer candidate or no longer running")
		r.resetVote()
		return nil
	}

	result, err := r.requestVotes()
	if err != nil {
		return err
	}

	if r.shouldStopElection() {
		r.debugf("should stop election; after request votes, no longer candidate or no longer running")
		r.resetVote()
		return nil
	}

	if result == ElectionVictory {
		r.debugf("election successful, promoting self to leader")
		r.setLeaderSafe()
		return r.heartbeat() // send immediate heartbeat
	}

	r.debugf("election loss or tie, backing off for rest of election period")

	// transition back to follower but do not reset the vote.
	r.transitionSafe(Follower)
	return nil
}

// requestVotes sends `RequestVote` rpcs to all peers, and totals the results.
func (r *Raft) requestVotes() (result ElectionOutcome, err error) {
	voteRequest := RequestVote{
		ID:   r.id,
		Term: r.currentTerm,
	}

	results := make(chan *RequestVoteResults, len(r.peers))
	errs := make(chan error, len(r.peers))
	wg := sync.WaitGroup{}
	wg.Add(len(r.peers))

	for _, peer := range r.peers {
		go func(c Client) {
			defer wg.Done()

			res, err := c.RequestVote(&voteRequest)
			if err != nil {
				r.debugf("seeking vote from %s: error", c.RemoteAddr())
				errs <- err
			} else {
				r.debugf("seeking vote from %s: %v", c.RemoteAddr(), res.Granted)
				results <- res
			}
		}(peer)
	}
	wg.Wait()
	r.logErrors(errs)

	result = r.processRequestVoteResults(results)
	r.debugf("election result: %v", result)
	return
}

func (r *Raft) sendHeartbeats() {
	args := AppendEntries{
		ID:   r.id,
		Term: r.currentTerm,
	}

	results := make(chan *AppendEntriesResults, len(r.peers))
	errs := make(chan error, len(r.peers))

	wg := sync.WaitGroup{}
	wg.Add(len(r.peers))
	for _, peer := range r.peers {
		go func(c Client) {
			defer wg.Done()
			res, err := c.AppendEntries(&args)
			if err != nil {
				errs <- err
			} else {
				results <- res
			}
		}(peer)
	}
	wg.Wait()

	r.logErrors(errs)

	// figure out how many rejections we got ...
	// if we didn't get a majority, demote self.
	if r.processAppendEntriesResults(results) != ElectionVictory {
		r.setFollowerSafe()
	}
}

// --------------------------------------------------------------------------------
// implementation methods
// --------------------------------------------------------------------------------

// processRequestVoteResults returns the aggregate votes for in an election from rpc responses.
func (r *Raft) processRequestVoteResults(results chan *RequestVoteResults) ElectionOutcome {
	// tabulate results
	total := len(r.peers) + 1 // assume cluster size is peers + 1 (ourselves)
	resultsCount := len(results)
	votesFor := 1 // assume we voted for ourselves.

	for index := 0; index < resultsCount; index++ {
		result := <-results

		if result.Granted {
			votesFor = votesFor + 1
		}
	}

	r.debugf("election tally: %d votes for, %d total (includes self)", votesFor, total)
	return r.voteOutcome(votesFor, total)
}

// processRequestVoteResults returns the aggregate votes for in an election from rpc responses.
func (r *Raft) processAppendEntriesResults(results chan *AppendEntriesResults) ElectionOutcome {
	// tabulate results
	total := len(r.peers) + 1 // assume cluster size is peers + 1 (ourselves)
	resultsCount := len(results)
	votesFor := 1 // assume we voted for ourselves.

	for index := 0; index < resultsCount; index++ {
		result := <-results

		if result.Success {
			votesFor = votesFor + 1
		}
	}

	r.debugf("heartbeat tally: %d votes for, %d total (includes self)", votesFor, total)
	return r.voteOutcome(votesFor, total)
}

// voteOutcome compares votes for to total and  it returns and integer
// indicating victory, tie, or loss. We assume both the votesFor and total
// do not include the implied self votes (you should add them before this step).
//  1 == victory
//  0 == tie
// -1 == loss
func (r *Raft) voteOutcome(votesFor, total int) ElectionOutcome {
	// if we have fewer than 2 responses, we can assume we're isolated, and treat it as a loss.
	if total < 2 {
		return ElectionLoss
	}

	majority := total >> 1
	// if we have an even total ...
	// this is the only situation where a vote can be a tie
	if total%2 == 0 {
		if votesFor > majority {
			return ElectionVictory
		} else if votesFor == majority {
			return ElectionTie
		}
		return ElectionLoss
	}

	// otherwise we can have a clear majority
	if votesFor > majority {
		return ElectionVictory
	}
	return ElectionLoss
}

func (r *Raft) transitionSafe(newState State) {
	r.Lock()
	defer r.Unlock()
	r.transition(newState)
}

// transition does a state transition, firing transition listeners.
func (r *Raft) transition(newState State) {
	isTransition := newState != r.state
	if isTransition {
		r.debugf("transitioning to %s", newState)
	}
	r.state = newState
	if isTransition {
		switch newState {
		case Follower:
			if r.followerHandler != nil {
				r.safeExecute(r.followerHandler)
			}
		case Candidate:
			if r.candidateHandler != nil {
				r.safeExecute(r.candidateHandler)
			}
		case Leader:
			if r.leaderHandler != nil {
				r.safeExecute(r.leaderHandler)
			}
		}
	}
}

func (r *Raft) shouldTriggerElection() (output bool) {
	r.Lock()
	now := time.Now().UTC()
	isLeaderTimedOut := r.lastLeaderContact.IsZero() || now.Sub(r.lastLeaderContact) > RandomTimeout(r.electionTimeout)
	isElectionDue := r.lastVoteGranted.IsZero() || now.Sub(r.lastVoteGranted) > RandomTimeout(r.electionTimeout)
	output = isLeaderTimedOut && isElectionDue
	r.Unlock()
	return
}

func (r *Raft) shouldStopElection() bool {
	return r.IsNotState(Candidate) || !r.latch.IsRunning()
}

func (r *Raft) resetVoteSafe() {
	r.Lock()
	defer r.Unlock()
	r.resetVote()
}

func (r *Raft) resetVote() {
	r.lastVoteGranted = time.Time{}
	r.votedFor = ""
}

func (r *Raft) voteSelf() {
	r.lastVoteGranted = r.now()
	r.votedFor = r.id
}

func (r *Raft) setFollowerSafe() {
	r.Lock()
	defer r.Unlock()
	r.setFollower()
}

func (r *Raft) setFollower() {
	r.transition(Follower)
	r.resetVote()
}

func (r *Raft) setCandidateSafe() {
	r.Lock()
	defer r.Unlock()
	r.setCandidate()
}

func (r *Raft) setCandidate() {
	r.currentTerm = r.currentTerm + 1
	r.transition(Candidate)
	r.voteSelf()
}

func (r *Raft) setLeaderSafe() {
	r.Lock()
	defer r.Unlock()
	r.setLeader()
}

func (r *Raft) setLeader() {
	r.transition(Leader)
	r.resetVote()
}

// now returns the current time in utc.
func (r *Raft) now() time.Time {
	return time.Now().UTC()
}

// --------------------------------------------------------------------------------
// runtime methods
// --------------------------------------------------------------------------------

func (r *Raft) safeExecute(action func()) {
	defer func() {
		if p := recover(); p != nil {
			r.err(fmt.Errorf("%+v", p))
		}
	}()
	action()
}

// --------------------------------------------------------------------------------
// logging methods
// --------------------------------------------------------------------------------

func (r *Raft) logErrors(errs chan error) {
	if errCount := len(errs); errCount > 0 {
		for index := 0; index < errCount; index++ {
			r.err(<-errs)
		}
	}
}

func (r *Raft) infof(format string, args ...interface{}) {
	if r.log != nil {
		r.log.SubContext("raft").SubContext(fmt.Sprintf("%v @ %d", r.State(), r.CurrentTerm())).Infof(format, args...)
	}
}

func (r *Raft) debugf(format string, args ...interface{}) {
	if r.log != nil {
		r.log.SubContext("raft").SubContext(fmt.Sprintf("%v @ %d", r.State(), r.CurrentTerm())).Debugf(format, args...)
	}
}

func (r *Raft) err(err error) {
	if r.log != nil && err != nil {
		r.log.SubContext("raft").SubContext(fmt.Sprintf("%v @ %d", r.State(), r.CurrentTerm())).Trigger(logger.Errorf(logger.Error, "%v", err))
	}
}
