package cluster

import (
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"sync"
	"time"
)

type Peer struct {
	info *control_pb.RegistryInfo

	buffer       *entriesBuffer
	mu           sync.RWMutex
	prevIndex    uint64
	lastActivity time.Time
	unavailable  bool
}

func (p *Peer) getIndexDelta(index uint64) uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return index - p.prevIndex
}
