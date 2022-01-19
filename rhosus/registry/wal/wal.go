package wal

import (
	"errors"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/wal_pb"
	"github.com/parasource/rhosus/rhosus/util/fileutil"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var (
	SegmentSizeBytes int64 = 64 * 1000 * 1000
)

type WAL struct {
	dir     string
	dirFile *os.File

	metadata  []byte
	start     *wal_pb.Snapshot
	lastIndex int64
	locks     []*fileutil.LockedFile

	decoder *decoder

	mu sync.RWMutex
}

func Create(dir string, metadata []byte) (*WAL, error) {
	if dirExists(dir) {
		return nil, errors.New("wal dir already exists")
	}

	tmpdirpath := filepath.Clean(dir) + ".tmp"
	if fileutil.Exist(tmpdirpath) {
		if err := os.RemoveAll(tmpdirpath); err != nil {
			return nil, err
		}
	}
	defer os.RemoveAll(tmpdirpath)

	if err := fileutil.CreateDirAll(tmpdirpath); err != nil {
		return nil, err
	}

	p := filepath.Join(tmpdirpath, walName(0, 0))
	f, err := fileutil.LockFile(p, os.O_WRONLY|os.O_CREATE, fileutil.PrivateFileMode)
	if err != nil {
		return nil, err
	}

	if _, err = f.Seek(0, io.SeekEnd); err != nil {
		return nil, err
	}

	if err = fileutil.Preallocate(f.File, SegmentSizeBytes, true); err != nil {
		return nil, err
	}

	w := &WAL{
		dir:      dir,
		metadata: metadata,
	}
	w.locks = append(w.locks, f)

	return nil, err

}

func (w *WAL) ReadAll() (metadata []byte, entries []control_pb.Entry, err error) {

	w.mu.Lock()
	defer w.mu.Unlock()

	log := &wal_pb.Log{}

	if w.decoder == nil {
		return nil, nil, errors.New("decoder not found")
	}
	decoder := w.decoder

	var match bool
	for err = decoder.decode(log); err == nil; err = decoder.decode(log) {
		switch log.Type {
		case wal_pb.Log_TYPE_ENTRY:

			var e control_pb.Entry
			err = e.Unmarshal(log.Data)
			if err != nil {
				// todo: this should never happen
				return nil, nil, errors.New("error unmarshaling")
			}

			if e.Index > w.start.Index {
				// prevent "panic: runtime error: slice bounds out of range [:13038096702221461992] with capacity 0"
				up := e.Index - w.start.Index - 1
				if up > int64(len(entries)) {
					// return error before append call causes runtime panic
					return nil, nil, errors.New("slice out of range")
				}
				// The line below is potentially overriding some 'uncommitted' entries.
				entries = append(entries[:up], e)
			}
			w.lastIndex = e.Index

		case wal_pb.Log_TYPE_SNAPSHOT:

			var snap wal_pb.Snapshot
			err := snap.Unmarshal(log.Data)
			if err != nil {
				// todo: this should never happen
				return nil, nil, errors.New("error unmarshaling")
			}

			if snap.Index == w.start.Index {
				if snap.Term != w.start.Term {
					return nil, nil, errors.New("snapshot mismatch")
				}
				match = true
			}

		}
	}

	err = nil
	if !match {
		err = errors.New("snapshot not found")
	}

	w.start = &wal_pb.Snapshot{}
	w.metadata = metadata

	return metadata, entries, err

}
