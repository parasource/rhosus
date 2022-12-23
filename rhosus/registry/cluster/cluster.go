package cluster

import (
	"errors"
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/registry/wal"
	"github.com/parasource/rhosus/rhosus/util/timers"
	"github.com/sirupsen/logrus"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const ElectionTimeoutThresholdPercent = 0.8

var (
	ErrShutdown     = errors.New("cluster is shut down")
	ErrPeerNotFound = errors.New("peer not found")
	ErrWriteTimeout = errors.New("cluster write timeout")
)

type Config struct {
	ID          string
	WalPath     string
	ClusterAddr string

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

	server       *ControlServer
	service      *ControlService
	wal          *wal.WAL
	RegistryInfo *control_pb.RegistryInfo

	writeEntriesC chan *writeRequest

	currentTerm  uint32
	lastLogIndex uint64
	lastLogTerm  uint32
	state        control_pb.State

	entriesAppendedC chan struct{}

	handleEntries func(entries []*control_pb.Entry)

	shutdown  bool
	shutdownC chan struct{}
	readyC    chan struct{}
}

func NewCluster(config Config, peers map[string]*control_pb.RegistryInfo) (*Cluster, error) {

	c := &Cluster{

		ID:     config.ID,
		config: config,

		peers: make(map[string]*Peer),

		writeEntriesC: make(chan *writeRequest),

		currentTerm:  0,
		lastLogIndex: 0,
		state:        control_pb.State_FOLLOWER,

		// just magic number
		entriesAppendedC: make(chan struct{}, 50),

		shutdownC: make(chan struct{}),
		readyC:    make(chan struct{}, 1),
	}

	// First, we create a Write-ahead Log
	walPath := config.WalPath
	err := os.Mkdir(walPath, 0755)
	if os.IsExist(err) {
		// triggers if dir already exists
		logrus.Debugf("backend path already exists, skipping")
	} else if err != nil {
		return nil, fmt.Errorf("error creating backend folder in %v: %v", walPath, err)
	}

	w, err := wal.Create(walPath, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating wal: %w", err)
	}
	c.wal = w
	c.setInitialTermAndIndex()

	// Server to accept other conns connections
	clusterAddr := config.ClusterAddr
	server, err := NewControlServer(c, clusterAddr)
	if err != nil {
		logrus.Fatalf("error creating control server: %v", err)
	}
	logrus.Infof("listening control on %v", clusterAddr)

	// Creating a service that holds connections to other conns
	addresses := make(map[string]string, len(peers))
	for uid, info := range peers {
		// we skip ourselves
		if uid == c.ID {
			continue
		}
		addresses[uid] = info.Address
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

	go func() {
		for {
			time.Sleep(time.Second * 2)

			if !c.isLeader() {
				continue
			}

			err := c.WriteEntries([]*control_pb.Entry{
				{
					Index:     c.lastLogIndex + 1,
					Term:      c.GetCurrentTerm(),
					Type:      control_pb.Entry_ASSIGN,
					Data:      []byte("ENTRY"),
					Timestamp: time.Now().Unix(),
				},
			})
			if err != nil {
				logrus.Errorf("error writing entries: %v", err)
			}
		}

	}()

	return c, nil
}

func (c *Cluster) SetRegistryInfo(info *control_pb.RegistryInfo) {
	c.RegistryInfo = info
}

func (c *Cluster) setInitialTermAndIndex() {
	var err error

	lastIndex, err := c.wal.LastIndex()
	if err != nil {
		logrus.Fatalf("error reading last entry in journal: %v", err)
	}

	if lastIndex == 0 {
		c.SetLastLogIndex(0)
		c.SetCurrentTerm(0)

		return
	}

	data, err := c.wal.Read(lastIndex)
	if err != nil {
		logrus.Fatalf("error reading last entry in journal: %v", err)
	}
	var entry control_pb.Entry
	entry.Unmarshal(data)

	c.SetLastLogIndex(entry.Index)
	c.SetLastLogTerm(entry.Term)
	c.SetCurrentTerm(entry.Term)
}

func (c *Cluster) SetEntriesHandler(f func(entries []*control_pb.Entry)) {
	c.mu.Lock()
	c.handleEntries = f
	c.mu.Unlock()
}

func (c *Cluster) SetLastLogTerm(term uint32) {
	c.mu.Lock()
	c.lastLogTerm = term
	c.mu.Unlock()
}

func (c *Cluster) SetLastLogIndex(index uint64) {
	c.mu.Lock()
	c.lastLogIndex = index
	c.mu.Unlock()
}

func (c *Cluster) SetCurrentTerm(term uint32) {
	c.mu.Lock()
	c.currentTerm = term
	c.mu.Unlock()
}

func (c *Cluster) GetCurrentTerm() uint32 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.currentTerm
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
		err = c.service.AddPeer(uid, info.Address)
		if err != nil {
			return err
		}

		c.peers[uid] = &Peer{
			info:               info,
			buffer:             &entriesBuffer{},
			lastCommittedIndex: 0,
			lastActivity:       time.Now(),
			recovering:         false,
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

//func (c *Cluster) IsWalEmpty() bool {
//	return c.wal.isEmpty()
//}

//func (c *Cluster) isPromotable() bool {
//	lastIndex, err := c.wal.LastIndex()
//	if err != nil {
//		return false
//	}
//
//	return lastIndex > 0
//}

func (c *Cluster) MemberCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// we include ourselves
	return len(c.peers) + 1
}

func (c *Cluster) Run() {

	go c.LoopWriteEntries()
	go c.LoopReadEntries()

	c.WritePipeline()
}

// LoopWriteEntries writes new entries to
// followers
func (c *Cluster) LoopWriteEntries() {

	for {
		c.mu.RLock()
		if c.shutdown {
			c.mu.RUnlock()
			return
		}
		c.mu.RUnlock()

		// if observer doesn't hear from leader in 150 to 300 ms, current peer becomes a candidate
		// and starts an election
		heartbeatTick := timers.SetTimer(750 * time.Millisecond)

		if <-heartbeatTick.C; true {

			c.mu.RLock()
			if !c.isLeader() {
				c.mu.RUnlock()
				continue
			}
			c.mu.RUnlock()

			c.dispatchEntries()
		}
	}
}

// LoopReadEntries listens for incoming entries
// from the leader. If it does not hear from leader
// for some time, it will start a new voting
func (c *Cluster) LoopReadEntries() {

	for {
		c.mu.RLock()
		if c.shutdown {
			c.mu.RUnlock()
			return
		}
		c.mu.RUnlock()

		// We need to set random interval between 150 and 300 ms
		// It is something similar to RAFT, but not completely
		electionTimout := timers.SetTimer(getIntervalMs(1500, 2000))

		select {
		// Leader is alright, so we discard timeout
		case <-c.entriesAppendedC:

			logrus.Info("HEARD FROM LEADER")

			timers.ReleaseTimer(electionTimout)

		case <-electionTimout.C:

			timers.ReleaseTimer(electionTimout)

			c.mu.RLock()
			if c.isLeader() {
				c.mu.RUnlock()
				continue
			}
			c.mu.RUnlock()

			err := c.StartElection()
			if err != nil {
				logrus.Errorf("error starting election proccess: %v", err)
			}
		}
	}
}

func (c *Cluster) dispatchEntries() {

	peers := make(map[string]*Peer)

	c.mu.RLock()
	for uid, peer := range c.peers {
		peers[uid] = peer
	}
	c.mu.RUnlock()

	for uid, peer := range peers {

		// If the peer is currently recovering, we don't need
		// to send him new entries, we just stack them in its buffer
		if peer.IsRecovering() {
			continue
		}

		go func(uid string, peer *Peer) {

			entries := peer.buffer.Read()
			req := &control_pb.AppendEntriesRequest{
				Term:         c.currentTerm,
				LeaderId:     c.ID,
				Entries:      entries,
				LeaderCommit: true,
			}

			res, err := c.service.AppendEntries(uid, req)
			if err != nil {
				//logrus.Warnf("error appending entries: %v", err)
				return
			}

			// In this case we need to retry with less index
			if !res.Success {
				err := c.startLogRecovering(uid, res.LastCommittedIndex)
				if err != nil {
					logrus.Errorf("error starting recovering proccess: %v", err)
				}
			}

		}(uid, peer)
	}
}

func (c *Cluster) startLogRecovering(uid string, from uint64) error {

	c.mu.RLock()
	if peer, ok := c.peers[uid]; ok {
		c.mu.RUnlock()

		peer.SetRecovering(true)
		defer peer.SetRecovering(false)

		lastIndex := c.lastLogIndex

		for index := from + 1; index <= lastIndex; index++ {

			bytes, err := c.wal.Read(index)
			if err != nil {
				logrus.Errorf("error getting entry while recovering peer: %v", err)
				return err
			}

			var entry control_pb.Entry
			err = entry.Unmarshal(bytes)
			if err != nil {
				logrus.Errorf("error unmarshaling entry: %v", err)
			}

			res, err := c.service.AppendEntries(uid, &control_pb.AppendEntriesRequest{
				Term:         c.currentTerm,
				LeaderId:     c.ID,
				Entries:      []*control_pb.Entry{&entry},
				LeaderCommit: true,
			})
			if err != nil {
				return err
			}
			// This should NEVER happen
			if !res.Success {
				return errors.New("unexpected error while recovering peer's wal")
			}

			peer.SetLastCommittedIndex(index)

		}

		return nil
	}
	c.mu.RUnlock()

	return ErrPeerNotFound
}

func (c *Cluster) StartElection() error {

	logrus.Info("STARTING ELECTION")

	c.becomeCandidate()

	c.mu.Lock()
	c.currentTerm++
	var peers []*Peer
	for _, peer := range c.peers {
		peers = append(peers, peer)
	}
	c.mu.Unlock()

	logrus.Infof("ELECTION started for %v peers at term %v", len(c.peers), c.GetCurrentTerm())

	var voted, responded int32
	var wg sync.WaitGroup

	for _, peer := range peers {
		wg.Add(1)
		go func(uid string) {
			defer wg.Done()

			res, err := c.service.sendVoteRequest(uid)
			if err != nil {
				//logrus.Warnf("error sending vote request to %v: %v", uid, err)
				return
			}

			atomic.AddInt32(&responded, 1)

			if res.VoteGranted {
				atomic.AddInt32(&voted, 1)
			}
		}(peer.info.Id)
	}
	wg.Wait()

	if responded < 1 || float64(voted) >= math.Floor(float64(responded/2)) {
		c.becomeLeader()
		logrus.Debug("changed node state to a leader")
	}

	return nil
}

// becomeFollower changes node state to follower
func (c *Cluster) becomeFollower() {
	c.mu.Lock()
	c.state = control_pb.State_FOLLOWER
	c.mu.Unlock()
}

// becomeLeader changes node state to leader
func (c *Cluster) becomeLeader() {
	c.mu.Lock()
	c.state = control_pb.State_LEADER
	c.mu.Unlock()
}

// becomeCandidate changes node state to candidate
func (c *Cluster) becomeCandidate() {
	c.mu.Lock()
	c.state = control_pb.State_CANDIDATE
	c.mu.Unlock()
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

func (c *Cluster) Shutdown() {
	close(c.shutdownC)

	c.mu.Lock()
	c.shutdown = true
	c.mu.Unlock()
}
