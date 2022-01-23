package registry

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/pb/wal_pb"
	"github.com/parasource/rhosus/rhosus/registry/wal"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/timers"
	"github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"time"
)

type Term struct {
	votes map[string]uint32
}

// Observer watches and starts and election if there is no signal within an interval
type Observer struct {
	registry *Registry

	initiateVotingCh   chan struct{}
	heartbeatTimeoutMs int

	server  *ControlServer
	service *ControlService
	wal     *wal.WAL

	readyC chan struct{}
}

func NewObserver(registry *Registry) *Observer {

	w, err := wal.Create("rhosuswal", []byte("test wal"))
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
	service, err := NewControlService(registry, map[string]ServerAddress{})
	if err != nil {
		logrus.Fatalf("error creating control service: %v", err)
	}

	for i := 0; i < 100; i++ {
		err = w.Encode(&wal_pb.Log{
			Type: wal_pb.Log_TYPE_ENTRY,
			Data: []byte(util.GenerateRandomName(2)),
		})
		if err != nil {
			logrus.Fatalf("error encoding log: %v", err)
		}
		err = w.Flush()
		if err != nil {
			logrus.Errorf("error flushing log: %v", err)
		}
	}

	o := &Observer{
		initiateVotingCh: make(chan struct{}, 1),
		registry:         registry,

		wal:     w,
		server:  server,
		service: service,

		readyC: make(chan struct{}, 1),
	}

	return o
}

func (o *Observer) Observe() {
	for {
		// According to RAFT docs, we need to set random interval
		// between 150 and 300 ms
		timer := timers.SetTimer(time.Millisecond * getInterval())

		// if observer doesn't hear from leader in 100 ms, current peer becomes a candidate
		timeout := timers.SetTimer(time.Millisecond * time.Duration(o.heartbeatTimeoutMs))

		select {
		case <-timer.C:
			timers.ReleaseTimer(timer)
			timers.ReleaseTimer(timeout)
			// everything is alright, leader is up and running
		case <-timeout.C:
			timers.ReleaseTimer(timer)
			timers.ReleaseTimer(timeout)

			o.initiateVotingCh <- struct{}{}
		}
	}
}

func (o *Observer) NotifyStartVoting() <-chan struct{} {
	return o.initiateVotingCh
}

func getInterval() time.Duration {
	return time.Duration(rand.Intn(300-150) + 150)
}
