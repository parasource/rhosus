package cluster

import (
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"sync"
	"time"
)

type Peer struct {
	info *control_pb.RegistryInfo

	buffer             *entriesBuffer
	mu                 sync.RWMutex
	lastCommittedIndex uint64
	lastActivity       time.Time

	recovering bool
}

func (p *Peer) IsRecovering() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.recovering
}

func (p *Peer) SetRecovering(recovering bool) {
	p.mu.Lock()
	p.recovering = recovering
	p.mu.Unlock()
}

func (p *Peer) SetLastCommittedIndex(index uint64) {
	p.mu.Lock()
	p.lastCommittedIndex = index
	p.mu.Unlock()
}
