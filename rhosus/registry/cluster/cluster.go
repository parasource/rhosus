package cluster

import (
	"errors"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/registry/wal"
	"github.com/parasource/rhosus/rhosus/util/timers"
	"github.com/sirupsen/logrus"
	"net"
	_ "net/http/pprof"
	"sync"
	"time"
)

const ElectionTimeoutThresholdPercent = 0.8

var (
	ErrShutdown = errors.New("cluster is shut down")
)

type Config struct {
	ID string

	RegistryInfo *control_pb.RegistryInfo

	HeartbeatIntervalMinMs int
	HeartbeatIntervalMaxMs int

	ElectionTimeoutMinMs int
	ElectionTimeoutMaxMs int

	MaxEntriesPerRequest int
}

// Cluster watches and starts and election if there is no signal within an interval
type Cluster struct {
	config Config

	ID string

	mu          sync.RWMutex
	peers       map[string]*Peer
	syncedPeers map[string]bool
	leader      string

	server  *ControlServer
	service *ControlService
	wal     *wal.WAL

	entriesC chan *control_pb.Entry
	buffer   *entriesBuffer

	currentTerm  uint32
	lastLogIndex uint64
	state        control_pb.State

	electionTimeoutC chan struct{}
	// When we change a state of a node to leader of follower,
	// we need to stop corresponding goroutines
	// to prevent busy loop
	stopSendingC  chan struct{}
	stopWatchingC chan struct{}

	shutdown  bool
	shutdownC chan struct{}
	readyC    chan struct{}
}

func NewCluster(config Config, peers map[string]*control_pb.RegistryInfo) *Cluster {

	c := &Cluster{

		ID:     config.ID,
		config: config,

		peers: make(map[string]*Peer),

		entriesC: make(chan *control_pb.Entry),
		buffer:   &entriesBuffer{},

		currentTerm:  0,
		lastLogIndex: 0,
		state:        control_pb.State_FOLLOWER,

		stopSendingC:     make(chan struct{}, 1),
		stopWatchingC:    make(chan struct{}, 1),
		electionTimeoutC: make(chan struct{}, 1),

		shutdownC: make(chan struct{}),
		readyC:    make(chan struct{}, 1),
	}

	// First, we create a Write-ahead Log
	w, err := wal.Create("wal", nil)
	if err != nil {
		logrus.Fatalf("error creating wal: %v", err)
		return nil
	}
	c.wal = w

	// Server to accept other conns connections
	srvAddress := net.JoinHostPort(config.RegistryInfo.Address.Host, config.RegistryInfo.Address.Port)
	server, err := NewControlServer(c, srvAddress)
	if err != nil {
		logrus.Fatalf("error creating control server: %v", err)
	}
	logrus.Infof("listening control on %v", srvAddress)

	// Creating a service that holds connections to other conns
	addresses := make(map[string]string, len(peers))
	for uid, info := range peers {
		// we skip ourselves
		if uid == c.ID {
			continue
		}
		addresses[uid] = composeRegistryAddress(info.Address)
	}
	service, sanePeers, err := NewControlService(c, addresses)
	if err != nil {
		logrus.Fatalf("error creating control service: %v", err)
	}

	c.wal = w
	c.server = server
	c.service = service

	// We add only working conns to our map
	// We will try to reconnect to others some time later
	for _, uid := range sanePeers {
		c.peers[uid] = &Peer{
			info:   peers[uid],
			buffer: &entriesBuffer{},
		}
	}

	// At this point there are no other registries, so we promote us to a leader
	if len(c.peers) < 1 {
		c.state = control_pb.State_LEADER
	}

	go c.Run()

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
	if _, ok := c.peers[uid]; ok {

		//TODO

	} else {
		err = c.service.AddPeer(uid, composeRegistryAddress(info.Address))
		if err != nil {
			return err
		}

		c.peers[uid] = &Peer{
			info:         info,
			buffer:       &entriesBuffer{},
			prevIndex:    0,
			lastActivity: time.Now(),
			unavailable:  false,
		}

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

//func (c *Cluster) IsWalEmpty() bool {
//	return c.wal.isEmpty()
//}

func (c *Cluster) isPromotable() bool {
	lastIndex, err := c.wal.LastIndex()
	if err != nil {
		return false
	}

	return lastIndex > 0
}

func (c *Cluster) MemberCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// we include ourselves
	return len(c.peers) + 1
}

func (c *Cluster) Run() {

	if c.isLeader() {
		go c.RunSendEntries()
	} else {
		go c.WatchForEntries()
	}

	for {

		c.mu.RLock()
		if c.shutdown {
			c.mu.RUnlock()
			return
		}
		c.mu.RUnlock()

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

		c.mu.RLock()
		if c.shutdown {
			c.mu.RUnlock()
			return
		}
		c.mu.RUnlock()

		// if observer doesn't hear from leader in 150 to 300 ms, current peer becomes a candidate
		// and starts an election
		heartbeatTick := timers.SetTimer(getIntervalMs(150, 300))

		select {
		case <-c.stopSendingC:
			return
		case <-heartbeatTick.C:

			entries := c.buffer.Read()
			req := &control_pb.AppendEntriesRequest{
				Term:         1,
				LeaderUid:    c.ID,
				PrevLogIndex: 0,
				PrevLogTerm:  1,
				Entries:      entries,
				LeaderCommit: true,
			}
			resp := c.service.AppendEntries(req)

			logrus.Infof("responses: %v", resp)
		}
	}
}

func (c *Cluster) WatchForEntries() {

	for {

		c.mu.RLock()
		if c.shutdown {
			c.mu.RUnlock()
			return
		}
		c.mu.RUnlock()

		// We need to set random interval between 150 and 300 ms
		// It is something similar to RAFT, but not completely
		electionTimout := timers.SetTimer(getIntervalMs(300, 500))
		select {
		// Leader is alright, so we discard timeout
		case <-c.electionTimeoutC:
			timers.ReleaseTimer(electionTimout)

		case <-electionTimout.C:
			timers.ReleaseTimer(electionTimout)

			c.state = control_pb.State_CANDIDATE
			c.currentTerm++

			err := c.StartElection()
			if err != nil {
				logrus.Errorf("error starting election proccess: %v", err)
			}

		case <-c.stopWatchingC:
			return
		}
	}
}

func (c *Cluster) StartElection() error {

	logrus.Infof("election started")
	respC := c.service.sendVoteRequests()

	var responded, voted = 0, 0
	var wg sync.WaitGroup

	for i := 0; i <= cap(respC); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			if vote := <-respC; true {
				if vote.err != nil {
					responded++

					logrus.Errorf("error getting vote from a peer: %v", vote.err)
					return
				}
				if vote.res.VoteGranted {
					voted++
				}
			}
		}()
	}

	wg.Wait()

	// It means there are no nodes alive, so we promote ourselves to a leader
	if responded == 0 || voted > responded/2 {
		c.becomeLeader()
		return nil
	}

	return nil
}

func (c *Cluster) becomeFollower() {
	c.mu.Lock()
	c.state = control_pb.State_FOLLOWER
	c.mu.Unlock()

	// Now we stop sending entries and start listening them
	c.stopSendingC <- struct{}{}
	go c.WatchForEntries()
}

func (c *Cluster) becomeLeader() {
	c.mu.Lock()
	c.state = control_pb.State_LEADER
	c.mu.Unlock()

	// Now we don't want node to listen for other entries
	c.stopWatchingC <- struct{}{}
	go c.RunSendEntries()
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
