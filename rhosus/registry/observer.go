package registry

import (
	"github.com/parasource/rhosus/rhosus/registry/wal"
	"github.com/parasource/rhosus/rhosus/util/timers"
	"github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

type Term struct {
	votes map[string]uint32
}

// Observer watches and starts and election if there is no signal within an interval
type Observer struct {
	service            *ControlService
	initiateVotingCh   chan struct{}
	heartbeatTimeoutMs int

	wal *wal.WAL
}

func NewObserver(service *ControlService) *Observer {
	o := &Observer{
		service:          service,
		initiateVotingCh: make(chan struct{}, 1),
	}

	w, err := wal.Create(".", []byte("aaaa blyaaat"))
	if err != nil {
		logrus.Fatalf("error creating wal: %v", err)
		return nil
	}
	o.wal = w

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
