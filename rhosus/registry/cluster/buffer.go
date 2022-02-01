package cluster

import (
	"errors"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/sirupsen/logrus"
	"sync"
)

var (
	ErrCorrupt = errors.New("corrupt entry index")
)

type entriesBuffer struct {
	mu      sync.RWMutex
	entries []*control_pb.Entry
}

func (b *entriesBuffer) Write(entry *control_pb.Entry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.entries[len(b.entries)-1].Index >= entry.Index {
		return ErrCorrupt
	}

	if b.entries[len(b.entries)-1].Term > entry.Term {
		logrus.Errorf("skipping entry with less term")
		return nil
	}

	b.entries = append(b.entries, entry)

	return nil
}

func (b *entriesBuffer) WriteBatch(entries []control_pb.Entry) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, entry := range entries {

		if b.entries[len(b.entries)-1].Index >= entry.Index {
			return ErrCorrupt
		}

		if b.entries[len(b.entries)-1].Term > entry.Term {
			logrus.Errorf("skipping entry with less term")
			continue
		}

		entries = append(entries, entry)
	}

	return nil
}

func (b *entriesBuffer) Read() []*control_pb.Entry {
	b.mu.Lock()
	defer b.mu.Unlock()

	entries := b.entries

	// clear the buffer
	b.entries = nil

	return entries
}
