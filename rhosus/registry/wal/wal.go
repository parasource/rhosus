package wal

import (
	"bytes"
	"errors"
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/wal_pb"
	"github.com/parasource/rhosus/rhosus/util/fileutil"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// A lot of code is brought from etcd source
//

const (

	// warnSyncDuration is the amount of time allotted to an fsync before
	// logging a warning
	warnSyncDuration = time.Second
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

	encoder   *encoder
	decoder   *decoder
	readClose func() error

	fp *filePipeline

	mu sync.RWMutex
}

// Create creates wal instance
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

	buf := new(bytes.Buffer)
	w := &WAL{
		dir:      dir,
		metadata: metadata,

		decoder: newDecoder(io.NopCloser(buf)),
	}
	w.encoder, err = newFileEncoder(f.File, 0)
	if err != nil {
		return nil, err
	}
	w.locks = append(w.locks, f)
	if err = w.saveCrc(0); err != nil {
		return nil, err
	}

	if err = w.encoder.encode(&wal_pb.Log{Type: wal_pb.Log_TYPE_ENTRY, Data: metadata}); err != nil {
		return nil, err
	}

	if w, err = w.renameWAL(tmpdirpath); err != nil {
		logrus.Fatalf("failed to rename tmp WAL directory: %v", err)
		return nil, err
	}

	var perr error
	defer func() {
		if perr != nil {
			w.cleanupWAL()
		}
	}()

	pdir, perr := fileutil.OpenDir(filepath.Dir(w.dir))
	if perr != nil {
		logrus.Fatalf("failed to open parent data dir: %v", err)
		return nil, perr
	}
	dirCloser := func() error {
		if perr = pdir.Close(); perr != nil {
			logrus.Errorf("failed to close parent data dir: %v", err)
			return perr
		}
		return nil
	}
	if perr = fileutil.Fsync(pdir); perr != nil {
		dirCloser()
		logrus.Fatalf("failed to sync parent data dir: %v", err)
		return nil, perr
	}
	if err = dirCloser(); err != nil {
		return nil, err
	}

	return w, nil

}

func (w *WAL) Encode(log *wal_pb.Log) error {
	return w.encoder.encode(log)
}

func (w *WAL) Flush() error {
	return w.encoder.flush()
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

func Open(dirpath string, snap wal_pb.Snapshot) (*WAL, error) {
	w, err := openAtIndex(dirpath, snap, true)
	if err != nil {
		return nil, err
	}
	if w.dirFile, err = fileutil.OpenDir(w.dir); err != nil {
		return nil, err
	}
	return w, nil
}

// OpenForRead only opens the wal files for read.
// Write on a read only wal panics.
func OpenForRead(dirpath string, snap wal_pb.Snapshot) (*WAL, error) {
	return openAtIndex(dirpath, snap, false)
}

func openAtIndex(dirpath string, snap wal_pb.Snapshot, write bool) (*WAL, error) {

	names, nameIndex, err := selectWALFiles(dirpath, snap)
	if err != nil {
		return nil, err
	}

	rs, ls, closer, err := openWALFiles(dirpath, names, nameIndex, write)
	if err != nil {
		return nil, err
	}

	// create a WAL ready for reading
	w := &WAL{
		dir:       dirpath,
		start:     &snap,
		decoder:   newDecoder(rs...),
		readClose: closer,
		locks:     ls,
	}

	if write {
		// write reuses the file descriptors from read; don't close so
		// WAL can append without dropping the file lock
		w.readClose = nil
		if _, _, err := parseWALName(filepath.Base(w.tail().Name())); err != nil {
			closer()
			return nil, err
		}
	}

	return w, nil
}

func selectWALFiles(dirpath string, snap wal_pb.Snapshot) ([]string, int, error) {
	names, err := readWALNames(dirpath)
	if err != nil {
		return nil, -1, err
	}

	nameIndex, ok := searchIndex(names, snap.Index)
	if !ok || !isValidSeq(names[nameIndex:]) {
		err = errors.New("file not found")
		return nil, -1, err
	}

	return names, nameIndex, nil
}

func openWALFiles(dirpath string, names []string, nameIndex int, write bool) ([]io.Reader, []*fileutil.LockedFile, func() error, error) {

	rcs := make([]io.ReadCloser, 0)
	rs := make([]io.Reader, 0)
	ls := make([]*fileutil.LockedFile, 0)
	for _, name := range names[nameIndex:] {
		p := filepath.Join(dirpath, name)
		if write {
			l, err := fileutil.TryLockFile(p, os.O_RDWR, fileutil.PrivateFileMode)
			if err != nil {
				closeAll(rcs...)
				return nil, nil, nil, err
			}
			ls = append(ls, l)
			rcs = append(rcs, l)
		} else {
			rf, err := os.OpenFile(p, os.O_RDONLY, fileutil.PrivateFileMode)
			if err != nil {
				closeAll(rcs...)
				return nil, nil, nil, err
			}
			ls = append(ls, nil)
			rcs = append(rcs, rf)
		}
		rs = append(rs, rcs[len(rcs)-1])
	}

	closer := func() error { return closeAll(rcs...) }

	return rs, ls, closer, nil
}

func (w *WAL) cleanupWAL() {
	var err error
	if err = w.Close(); err != nil {
		logrus.Fatalf("failed to close WAL during cleanup: %v", err)
	}
	brokenDirName := fmt.Sprintf("%s.broken.%v", w.dir, time.Now().Format("20060102.150405.999999"))
	if err = os.Rename(w.dir, brokenDirName); err != nil {
		logrus.Fatalf("failed to rename WAL during cleanup: %v", err)
	}
}

func (w *WAL) renameWAL(tmpdirpath string) (*WAL, error) {
	if err := os.RemoveAll(w.dir); err != nil {
		return nil, err
	}
	// On non-Windows platforms, hold the lock while renaming. Releasing
	// the lock and trying to reacquire it quickly can be flaky because
	// it's possible the process will fork to spawn a process while this is
	// happening. The fds are set up as close-on-exec by the Go runtime,
	// but there is a window between the fork and the exec where another
	// process holds the lock.
	if err := os.Rename(tmpdirpath, w.dir); err != nil {
		if _, ok := err.(*os.LinkError); ok {
			return w.renameWALUnlock(tmpdirpath)
		}
		return nil, err
	}
	w.fp = newFilePipeline(w.dir, SegmentSizeBytes)
	df, err := fileutil.OpenDir(w.dir)
	w.dirFile = df
	return w, err
}

func (w *WAL) renameWALUnlock(tmpdirpath string) (*WAL, error) {
	// rename of directory with locked files doesn't work on windows/cifs;
	// close the WAL to release the locks so the directory can be renamed.

	w.Close()

	if err := os.Rename(tmpdirpath, w.dir); err != nil {
		return nil, err
	}

	// reopen and relock
	newWAL, oerr := Open(w.dir, wal_pb.Snapshot{})
	if oerr != nil {
		return nil, oerr
	}
	if _, _, err := newWAL.ReadAll(); err != nil {
		newWAL.Close()
		return nil, err
	}
	return newWAL, nil
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	//if w.fp != nil {
	//	w.fp.Close()
	//	w.fp = nil
	//}

	if w.tail() != nil {
		if err := w.sync(); err != nil {
			return err
		}
	}
	for _, l := range w.locks {
		if l == nil {
			continue
		}
		if err := l.Close(); err != nil {
			//w.lg.Error("failed to close WAL", zap.Error(err))
		}
	}

	return w.dirFile.Close()
}

func closeAll(rcs ...io.ReadCloser) error {
	stringArr := make([]string, 0)
	for _, f := range rcs {
		if err := f.Close(); err != nil {
			stringArr = append(stringArr, err.Error())
		}
	}
	if len(stringArr) == 0 {
		return nil
	}
	return errors.New(strings.Join(stringArr, ", "))
}

func (w *WAL) tail() *fileutil.LockedFile {
	if len(w.locks) > 0 {
		return w.locks[len(w.locks)-1]
	}
	return nil
}

func (w *WAL) seq() int64 {
	t := w.tail()
	if t == nil {
		return 0
	}
	seq, _, err := parseWALName(filepath.Base(t.Name()))
	if err != nil {
		logrus.Errorf("failed to parse wal name")
	}
	return seq
}

func (w *WAL) sync() error {
	//if w.encoder != nil {
	//	if err := w.encoder.flush(); err != nil {
	//		return err
	//	}
	//}

	start := time.Now()
	err := fileutil.Fdatasync(w.tail().File)

	took := time.Since(start)
	if took > 5 {
		//	TODO: log slow datasync
	}

	return err
}

func (w *WAL) saveCrc(prevCrc uint32) error {
	return w.encoder.encode(&wal_pb.Log{Type: wal_pb.Log_TYPE_CRC, Crc: prevCrc})
}
