package cluster

import (
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util/timers"
	"github.com/sirupsen/logrus"
	"time"
)

type writeRequest struct {
	entries []*control_pb.Entry
	resp    chan *writeResponse
}

func (w *writeRequest) done(err error) {
	w.resp <- &writeResponse{err: err}
}

func (w *writeRequest) result() error {
	res := <-w.resp

	return res.err
}

type writeResponse struct {
	err error
}

func newWriteRequest(entries []*control_pb.Entry) *writeRequest {
	return &writeRequest{
		entries: entries,
		resp:    make(chan *writeResponse, 1),
	}
}

func (c *Cluster) WritePipeline() {

	for req := range c.writeEntriesC {

		c.mu.RLock()
		if c.shutdown {
			c.mu.RUnlock()
			return
		}
		c.mu.RUnlock()

		for _, entry := range req.entries {
			bytes, _ := entry.Marshal()
			err := c.wal.Write(entry.Index, bytes)
			if err != nil {
				logrus.Errorf("error writing to wal: %v", err)
			}

			c.SetLastLogIndex(entry.Index)
			c.SetLastLogTerm(entry.Term)
		}

		peers := make(map[string]*Peer)
		c.mu.RLock()
		for uid, peer := range c.peers {
			peers[uid] = peer
		}
		c.mu.RUnlock()

		for _, peer := range peers {

			err := peer.WriteToBuffer(req.entries)
			if err != nil {
				req.done(err)
				return
			}

		}

		req.done(nil)
	}
}

func (c *Cluster) WriteEntries(entries []*control_pb.Entry) error {

	c.mu.RLock()
	if c.shutdown {
		c.mu.RUnlock()
		return ErrShutdown
	}
	c.mu.RUnlock()

	req := newWriteRequest(entries)
	select {
	case c.writeEntriesC <- req:
	default:
		writeTimeout := timers.SetTimer(time.Millisecond * 50)

		select {
		case c.writeEntriesC <- req:
		case <-writeTimeout.C:
			return ErrWriteTimeout
		}
	}

	return req.result()
}

// WriteEntriesFromLeader is called by registry to write entries to
// local version of WAL
func (c *Cluster) WriteEntriesFromLeader(entries []*control_pb.Entry) error {

	err := validateEntriesSequence(entries)
	if err != nil {
		return err
	}

	// First, we need to call registry callback to write entries to database
	c.mu.RLock()
	if c.handleEntries != nil {
		c.mu.RUnlock()

		c.handleEntries(entries)
	}
	c.mu.RUnlock()

	for _, entry := range entries {

		bytes, _ := entry.Marshal()

		err := c.wal.Write(entry.Index, bytes)
		if err != nil {
			return err
		}

		c.SetLastLogIndex(entry.Index)
		c.SetLastLogTerm(entry.Term)
	}

	return nil

}

func validateEntriesSequence(entries []*control_pb.Entry) error {
	for i, entry := range entries {
		// First el is always alright
		if i == 0 {
			continue
		}

		if entry.Index <= entries[i-1].Index {
			return ErrCorrupt
		}
	}

	return nil
}
