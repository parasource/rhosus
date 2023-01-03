package cluster

import (
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util/timers"
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

// WriteAssignFileEntry writes an assign entry to buffer,
// which will be distributed to other nodes
func (c *Cluster) WriteAssignFileEntry(info *control_pb.FileInfo) error {
	assignEntry := &control_pb.EntryAssignFile{
		NodeId: c.ID,
		File:   info,
	}
	entryBytes, err := assignEntry.Marshal()
	if err != nil {
		return fmt.Errorf("error marshaling assign entry to bytes: %v", err)
	}

	return c.writeEntry(control_pb.Entry_ASSIGN_FILE, entryBytes)
}

// WriteDeleteFileEntry writes a delete entry to buffer,
// which will be distributed to other nodes
func (c *Cluster) WriteDeleteFileEntry(info *control_pb.FileInfo) error {
	deleteEntry := &control_pb.EntryDeleteFile{
		NodeId: c.ID,
		File:   info,
	}
	entryBytes, err := deleteEntry.Marshal()
	if err != nil {
		return fmt.Errorf("error marshaling delete entry to bytes: %v", err)
	}

	return c.writeEntry(control_pb.Entry_DELETE_FILE, entryBytes)
}

// WriteAssignBlocksEntry writes an assign entry to buffer,
// which will be distributed to other nodes
func (c *Cluster) WriteAssignBlocksEntry(blocks []*control_pb.BlockInfo) error {
	assignEntry := &control_pb.EntryAssignBlocks{
		NodeId: c.ID,
		Blocks: blocks,
	}
	entryBytes, err := assignEntry.Marshal()
	if err != nil {
		return fmt.Errorf("error marshaling assign entry to bytes: %v", err)
	}

	return c.writeEntry(control_pb.Entry_ASSIGN_BLOCKS, entryBytes)
}

// WriteDeleteBlocksEntry writes a delete entry to buffer,
// which will be distributed to other nodes
func (c *Cluster) WriteDeleteBlocksEntry(blocks []string) error {
	deleteEntry := &control_pb.EntryDeleteBlocks{
		NodeId: c.ID,
		Blocks: blocks,
	}
	entryBytes, err := deleteEntry.Marshal()
	if err != nil {
		return fmt.Errorf("error marshaling delete entry to bytes: %v", err)
	}

	return c.writeEntry(control_pb.Entry_DELETE_BLOCKS, entryBytes)
}

// WriteEntry writes an entry to buffer, which
// will be distributed to other nodes
func (c *Cluster) WriteEntry(typ control_pb.Entry_Type, data []byte) error {
	return c.writeEntry(typ, data)
}

func (c *Cluster) writeEntry(typ control_pb.Entry_Type, data []byte) error {
	c.mu.RLock()
	if c.shutdown {
		c.mu.RUnlock()
		return ErrShutdown
	}
	c.mu.RUnlock()

	lastLogIndex := c.GetLastLogIndex()
	currentTerm := c.GetCurrentTerm()

	entry := &control_pb.Entry{
		Index:     lastLogIndex + 1,
		Term:      currentTerm,
		Type:      typ,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
	c.SetLastLogIndex(lastLogIndex + 1)
	c.SetLastLogTerm(currentTerm)

	// Writing to wal storage
	bytes, _ := entry.Marshal()
	err := c.wal.Write(entry.Index, bytes)
	if err != nil {
		// rolling back to prev index
		c.SetLastLogIndex(lastLogIndex)
		return fmt.Errorf("error writing entry to wal storage: %v", err)
	}

	req := newWriteRequest([]*control_pb.Entry{entry})
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

func (c *Cluster) writeEntries(entries []*control_pb.Entry) error {
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
	if c.entriesHandler != nil {
		c.entriesHandler(entries)
	}

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
