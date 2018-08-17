package raft

import (
	"testing"
	"time"

	"github.com/blend/go-sdk/assert"
	"github.com/blend/go-sdk/uuid"
)

func TestIntegrationBootstrap1(t *testing.T) {
	//assert := assert.New(t)

	//c := createCluster(1)
	//c.start()
	//c.waitElection(assert)
}

func TestIntegrationBootstrap3(t *testing.T) {
	// assert := assert.New(t)

	// createCluster(3)
	// start
	// assert transitions to leader
}

func TestIntegrationBootstrap5(t *testing.T) {
	// assert := assert.New(t)

	// createCluster(5)
	// start
	// assert transitions to leader
}

func TestIntegrationLeaderLoss(t *testing.T) {
	// assert := assert.New(t)

	// createCluster(5)
	// start
	// assert transitions to leader

	// kill leader
	// assert elects new leader
}

func TestIntegrationLeaderTransportFailure(t *testing.T) {
	// assert := assert.New(t)

	// createCluster(5)
	// start
	// assert transitions to leader

	// kill xport between leader and random node
	// assert election fails
	// assert node resumes as follower
	// assert leader stays the same
}

func integrationConfig() *Config {
	return &Config{
		ID:                  uuid.V4().String(),
		ElectionTimeout:     100 * time.Microsecond,
		HeartbeatInterval:   10 * time.Microsecond,
		LeaderCheckInterval: 10 & time.Microsecond,
	}
}

func createTestNode() *Raft {
	return New().
		WithID(uuid.V4().String()).
		WithLeaderCheckInterval(time.Microsecond).
		WithHeartbeatInterval(time.Microsecond).
		WithElectionTimeout(time.Microsecond).
		WithServer(NewMockServer())
}

func createCluster(nodeCount int) *cluster {
	if nodeCount <= 0 {
		return nil
	}
	if nodeCount == 1 {
		return &cluster{Nodes: []*Raft{createTestNode()}}
	}

	cluster := cluster{
		Election: make(chan *Raft),
	}

	// create all the nodes
	peers := make([]*Raft, nodeCount)
	for index := 0; index < nodeCount; index++ {
		node := createTestNode()
		node.SetLeaderHandler(func() {
			cluster.Election <- node
		})
		peers[index] = node
	}

	// cross wire all the nodes
	for i := 0; i < nodeCount; i++ {
		for j := 0; j < nodeCount; j++ {
			if i != j {
				xport := NewMockTransport(peers[j].ID(), peers[j].Server())
				cluster.addTransport(peers[j].ID(), peers[j].ID(), xport)
				peers[i].WithPeer(xport)
			}
		}
	}

	return &cluster
}

type cluster struct {
	Config     Config
	Nodes      []*Raft
	Transports map[string]map[string]RPCClient
	Election   chan *Raft
}

func (c *cluster) start() {
	for _, node := range c.Nodes {
		node.Start()
	}
}

func (c *cluster) stop() {
	for _, node := range c.Nodes {
		node.Stop()
	}
}

func (c *cluster) addTransport(from, to string, xport RPCClient) {
	if c.Transports == nil {
		c.Transports = map[string]map[string]RPCClient{}
	}
	if xports, has := c.Transports[from]; has {
		xports[to] = xport
	} else {
		c.Transports[from] = map[string]RPCClient{
			to: xport,
		}
	}
}

func (c *cluster) waitElection(assert *assert.Assertions) *Raft {
	alarm := time.NewTimer(c.Config.GetElectionTimeout())
	select {
	case <-alarm.C:
		assert.FailNow("election timeout")
	case r := <-c.Election:
		return r
	}
	return nil
}
