package cluster

import (
	"context"
	"errors"
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/wal_pb"
	"github.com/parasource/rhosus/rhosus/registry/wal"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/timers"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"sync"
	"time"
)

type Term struct {
	votes map[string]uint32
}

type Config struct {
	entriesBufferSize int

	heartbeatTimeoutMinMs int
	heartbeatTimeoutMaxMs int

	electionTimeoutMinMs int
	electionTimeoutMaxMs int
}

// Cluster watches and starts and election if there is no signal within an interval
type Cluster struct {
	config Config

	mu          sync.RWMutex
	isLeader    bool
	isCandidate bool

	initiateVotingCh   chan struct{}
	heartbeatTimeoutMs int

	server  *ControlServer
	service *ControlService
	wal     *wal.WAL

	entriesC chan *control_pb.Entry
	buffer   *entriesBuffer

	readyC chan struct{}
}

type PeerURLs []string

func NewCluster(config Config, peers PeerURLs) *Cluster {

	w, err := wal.Create("rhosuswal", nil)
	if err != nil {
		logrus.Fatalf("error creating wal: %v", err)
		return nil
	}

	srvPort, err := util.GetFreePort()
	if err != nil {
		logrus.Fatalf("error getting free port: %v", err)
	}

	srvAddress := net.JoinHostPort("localhost", fmt.Sprintf("%v", srvPort))
	server, err := NewControlServer(srvAddress)
	if err != nil {
		logrus.Fatalf("error creating control server: %v", err)
	}
	service, err := NewControlService(map[string]ServerAddress{})
	if err != nil {
		logrus.Fatalf("error creating control service: %v", err)
	}

	//logs := make(map[uint64][]byte)
	//for i := 101; i < 1000000; i++ {
	//	log := &wal_pb.Log{
	//		Type:  wal_pb.Log_TYPE_ENTRY,
	//		Crc:   0,
	//		Data:  []byte(util.GenerateRandomName(3)),
	//	}
	//	bytes, err := log.Marshal()
	//	if err != nil {
	//		logrus.Fatalf("error marshaling: %v", err)
	//	}
	//
	//	logs[uint64(i)] = bytes
	//}

	//err = w.WriteBatch(logs)
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = w.Sync()
	//if err != nil {
	//	logrus.Errorf("error flushing log: %v", err)
	//}

	bytes, err := w.Read(700001)
	if err != nil {
		logrus.Fatalf("error reading entry: %v", err)
	}

	var log wal_pb.Log
	err = log.Unmarshal(bytes)
	if err != nil {
		logrus.Fatalf("error unmarshaling entry: %v", err)
	}

	logrus.Infof("entry: %v", string(log.Data))

	o := &Cluster{
		config: config,

		initiateVotingCh: make(chan struct{}, 1),

		wal:     w,
		server:  server,
		service: service,

		entriesC: make(chan *control_pb.Entry),
		buffer:   &entriesBuffer{},

		readyC: make(chan struct{}, 1),
	}

	return o
}

func (c *Cluster) WriteEntry(entry *control_pb.Entry) error {

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

func (c *Cluster) RunSendEntries() {

	for {

		// Only leader should send entries and heartbeats
		if !c.isLeader {
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

				res, err := c.service.AppendEntries(ctx, req)
				if err != nil {

				}

			}()

		}
	}
}

func (c *Cluster) WatchForEntries() {
	for {

		// According to RAFT docs, we need to set random interval
		// between 150 and 300 ms
		electionTimout := timers.SetTimer(time.Millisecond * getIntervalMs(300, 500))

		select {
		case <-electionTimout.C:
			timers.ReleaseTimer(electionTimout)

			resp := c.service.sendVoteRequests()
			logrus.Infof("vote responses: %v", resp)
		}
	}
}

func (c *Cluster) NotifyStartVoting() <-chan struct{} {
	return c.initiateVotingCh
}

func getIntervalMs(min int, max int) time.Duration {
	return time.Millisecond * time.Duration(rand.Intn(max-min)+min)
}
