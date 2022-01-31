package cluster

import (
	"context"
	"errors"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/registry/wal"
	"github.com/parasource/rhosus/rhosus/util/timers"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

var (
	ErrShutdown = errors.New("cluster is shut down")
)

type Term struct {
	term     uint64
	votes    map[string]uint32
	votedFor uint32
}

type Config struct {
	RegistryInfo *control_pb.RegistryInfo

	HeartbeatTimeoutMinMs int
	HeartbeatTimeoutMaxMs int

	ElectionTimeoutMinMs int
	ElectionTimeoutMaxMs int
}

// Cluster watches and starts and election if there is no signal within an interval
type Cluster struct {
	config Config

	ID string

	mu    sync.RWMutex
	peers map[string]*control_pb.RegistryInfo

	server  *ControlServer
	service *ControlService
	wal     *wal.WAL

	entriesC chan *control_pb.Entry
	buffer   *entriesBuffer

	term         uint32
	lastLogIndex uint64
	votes        map[string]string
	state        control_pb.State

	shutdown  bool
	shutdownC chan struct{}
	readyC    chan struct{}
}

func NewCluster(config Config, peers map[string]*control_pb.RegistryInfo) *Cluster {

	c := &Cluster{
		config: config,

		peers: make(map[string]*control_pb.RegistryInfo),

		entriesC: make(chan *control_pb.Entry),
		buffer:   &entriesBuffer{},

		term:         0,
		lastLogIndex: 0,
		votes:        make(map[string]string),
		state:        control_pb.State_FOLLOWER,

		shutdownC: make(chan struct{}),
		readyC:    make(chan struct{}, 1),
	}

	// First, we create a Write-ahead Log
	w, err := wal.Create("wal", nil)
	if err != nil {
		logrus.Fatalf("error creating wal: %v", err)
		return nil
	}

	// Server to accept other peers connections
	srvAddress := net.JoinHostPort(config.RegistryInfo.Address.Host, config.RegistryInfo.Address.Port)
	server, err := NewControlServer(c, srvAddress)
	if err != nil {
		logrus.Fatalf("error creating control server: %v", err)
	}
	logrus.Infof("listening raft on %v", srvAddress)

	// Creating a service that holds connections to other peers
	addresses := make(map[string]string, len(peers))
	for uid, info := range peers {
		addresses[uid] = composeRegistryAddress(info.Address)
	}
	service, sanePeers, err := NewControlService(c, addresses)
	if err != nil {
		logrus.Fatalf("error creating control service: %v", err)
	}

	c.wal = w
	c.server = server
	c.service = service

	// We add only working peers to our map
	// We will try to reconnect to others some time later
	for _, uid := range sanePeers {
		c.peers[uid] = peers[uid]
	}

	return c
}

func (c *Cluster) SetLastLogIndex(index uint64) {
	c.mu.Lock()
	c.lastLogIndex = index
	c.mu.Unlock()
}

func (c *Cluster) DiscoverOrUpdate(uid string, info *control_pb.RegistryInfo) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.shutdown {
		return ErrShutdown
	}

	// Update existing peer if it exists
	if peer, ok := c.peers[uid]; ok {

		// TODO

	} else {
		err = c.service.AddPeer(uid, composeRegistryAddress(info.Address), false)
		if err != nil {
			return err
		}

		c.peers[uid] = peer

		logrus.Infof("added cluster peer %v with name %v", uid, info.Name)
	}

	return err
}

func (c *Cluster) RemovePeer(uid string) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.shutdown {
		return ErrShutdown
	}

	if _, ok := c.peers[uid]; ok {
		err = c.service.removePeer(uid)
	}

	return err
}

func (c *Cluster) WriteEntry(entry *control_pb.Entry) error {

	c.mu.RLock()
	if c.shutdown {
		c.mu.RUnlock()
		return ErrShutdown
	}
	c.mu.RUnlock()

	select {
	case c.entriesC <- entry:
	default:
		writeTimeout := timers.SetTimer(time.Millisecond * 500)

		select {
		case c.entriesC <- entry:
		case <-writeTimeout.C:
			return errors.New("entry write timeout")
		}
	}

	return nil
}

func (c *Cluster) Run() {

	go c.RunSendEntries()
	go c.WatchForEntries()

	for {

		if c.shutdown {
			return
		}

		select {
		case e := <-c.entriesC:
			err := c.buffer.Write(e)
			if err != nil {
				switch err {
				case ErrCorrupt:
					// TODO: do something
				}
			}
		}
	}

}

func (c *Cluster) Shutdown() {
	close(c.shutdownC)

	c.mu.Lock()
	c.shutdown = true
	c.mu.Unlock()
}

func (c *Cluster) RunSendEntries() {

	for {

		if c.shutdown {
			return
		}

		// Only leader should send entries and heartbeats
		if !c.isLeader() {
			continue
		}

		// if observer doesn't hear from leader in 100 ms, current peer becomes a candidate
		// and starts an election
		heartbeatTimeout := timers.SetTimer(time.Millisecond * getIntervalMs(150, 300))

		select {
		case <-heartbeatTimeout.C:

			entries := c.buffer.Read()
			req := &control_pb.AppendEntriesRequest{
				Term: 1,
				//LeaderUid:            "",
				PrevLogIndex: 0,
				PrevLogTerm:  1,
				Entries:      entries,
				LeaderCommit: true,
			}

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
				defer cancel()

				_, err := c.service.AppendEntries(ctx, req)
				if err != nil {

				}

			}()

		}
	}
}

func (c *Cluster) StartElection() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	respC := c.service.sendVoteRequests()

	var counter, votes = 0, 0

	for {
		select {
		case vote := <-respC:
			counter++

			if vote.res.Term > c.term {
				// step down
			}
			if vote.res.VoteGranted {
				votes++
			}

			if votes >= len(c.peers)/2+1 {
				// become leader

				break
			}
		}
	}
}

func (c *Cluster) becomeLeader() {
	c.state = control_pb.State_LEADER

}

func (c *Cluster) WatchForEntries() {

	for {

		if c.shutdown {
			return
		}

		// According to RAFT docs, we need to set random interval
		// between 150 and 300 ms
		electionTimout := timers.SetTimer(time.Millisecond * getIntervalMs(300, 500))

		select {
		case <-electionTimout.C:
			timers.ReleaseTimer(electionTimout)

			c.StartElection()
		}
	}
}

func (c *Cluster) NotifyShutdown() <-chan struct{} {
	return c.shutdownC
}

func (c *Cluster) isLeader() bool {
	return c.state == control_pb.State_LEADER
}

func (c *Cluster) isCandidate() bool {
	return c.state == control_pb.State_CANDIDATE
}

func (c *Cluster) isFollower() bool {
	return c.state == control_pb.State_FOLLOWER
}
