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

var (
	ErrShutdown = errors.New("cluster is shut down")
)

type Term struct {
	term     uint64
	votes    map[string]uint32
	votedFor uint32
}

type Config struct {
	ID string

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

	// When we change a state of a node to leader of follower,
	// we need to stop corresponding goroutines
	// to prevent busy loop
	stopSendingC         chan struct{}
	stopWatchingC        chan struct{}
	cancelVotingTimeoutC chan struct{}

	shutdown  bool
	shutdownC chan struct{}
	readyC    chan struct{}
}

func NewCluster(config Config, peers map[string]*control_pb.RegistryInfo) *Cluster {

	c := &Cluster{

		ID:     config.ID,
		config: config,

		peers: make(map[string]*control_pb.RegistryInfo),

		entriesC: make(chan *control_pb.Entry),
		buffer:   &entriesBuffer{},

		term:         0,
		lastLogIndex: 0,
		votes:        make(map[string]string),
		state:        control_pb.State_FOLLOWER,

		stopSendingC:         make(chan struct{}, 1),
		stopWatchingC:        make(chan struct{}, 1),
		cancelVotingTimeoutC: make(chan struct{}, 1),

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

	// We add only working peers to our map
	// We will try to reconnect to others some time later
	for _, uid := range sanePeers {
		c.peers[uid] = peers[uid]
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
		heartbeatTimeout := timers.SetTimer(getIntervalMs(150, 300))

		select {
		case <-c.stopSendingC:
			return
		case <-heartbeatTimeout.C:
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

		// According to RAFT docs, we need to set random interval
		// between 150 and 300 ms
		electionTimout := timers.SetTimer(getIntervalMs(300, 500))

		select {
		case <-c.cancelVotingTimeoutC:
			timers.ReleaseTimer(electionTimout)

		case <-electionTimout.C:
			timers.ReleaseTimer(electionTimout)

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

	respC := c.service.sendVoteRequests()

	var responded, voted = 0, 0

	// TODO: maybe add some workers
	for i := 0; i <= cap(respC); i++ {
		timeout := timers.SetTimer(time.Millisecond * 100)

		select {
		case <-timeout.C:
			// Node doesn't respond for too long, so we skip it
			// Actually we got to handle it more properly
			continue
		case vote := <-respC:
			if vote.err != nil {
				logrus.Errorf("error getting vote from a peer: %v", vote.err)
				continue
			}
			responded++

			//if vote.res.Term > c.term {
			//	// step down
			//}
			if vote.res.VoteGranted {
				voted++
			}
		}
	}

	// It means there are no nodes alive, so we promote ourselves to a leader
	if responded == 0 {
		c.becomeLeader()
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
