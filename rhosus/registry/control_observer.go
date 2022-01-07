package registry

import (
	"github.com/parasource/rhosus/rhosus/util/timers"
	"math/rand"
	"time"
)

type Observer struct {
	service *ControlService

	initiateVotingCh chan struct{}

	heartbeatTimeoutMs int
}

func NewObserver(service *ControlService) *Observer {
	return &Observer{
		service:          service,
		initiateVotingCh: make(chan struct{}, 1),
	}
}

func (o *Observer) Observe() {
	for {
		// According to RAFT docs, we need to set random interval
		// between 150 and 300 ms
		timer := timers.SetTimer(time.Millisecond * getRandomInterval())

		// if observer doesn't hear from leader for 100 ms, current peer becomes a candidate
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

func getRandomInterval() time.Duration {
	return time.Duration(rand.Intn(300-150) + 150)
}
