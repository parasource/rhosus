package backend

import (
	"errors"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/timers"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"sync"
	"time"
)

var (
	ErrWriteTimeout = errors.New("etcd storage write timeout")
	ErrShutdown     = errors.New("storage is shut down")
)

const (
	defaultDbFilePath = "indices.data"

	filesStorageBucketName  = "__files"
	blocksStorageBucketName = "__blocks"
	metaStorageBucketName   = "__meta"
)

type Config struct {
	DbFilePath string

	WriteTimeoutS int
	NumWorkers    int
}

type Storage struct {
	config Config
	db     *bolt.DB

	fileReqC   chan StoreReq
	blocksReqC chan StoreReq
	shutdownC  chan struct{}

	mu       sync.RWMutex
	shutdown bool
}

func NewStorage(config Config) (*Storage, error) {

	s := &Storage{
		config: config,

		fileReqC:   make(chan StoreReq),
		blocksReqC: make(chan StoreReq),
		shutdownC:  make(chan struct{}),
	}

	path := defaultDbFilePath
	if config.DbFilePath != "" {
		path = config.DbFilePath
	}
	db, err := bolt.Open(path, 0666, nil)
	if err != nil {
		return nil, err
	}
	s.db = db

	err = s.setup()
	if err != nil {
		logrus.Fatalf("error setting up backend: %v", err)
	}

	go s.start()

	go func() {
		if <-s.NotifyShutdown(); true {
			err := db.Close()
			if err != nil {
				logrus.Errorf("error closing bbolt: %v", err)
			}
			return
		}
	}()

	return s, nil
}

func (s *Storage) setup() (err error) {

	err = s.db.Update(func(tx *bolt.Tx) error {
		var err error

		_, err = tx.CreateBucketIfNotExists([]byte(filesStorageBucketName))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(blocksStorageBucketName))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(metaStorageBucketName))
		if err != nil {
			return err
		}

		return nil
	})

	return err
}

func (s *Storage) start() {

	go s.loopFileHandlers()
	go s.loopBlocksHandlers()

}

//////////////////////////////
// ---------------------------
// File methods
// ---------------------------

func (s *Storage) StoreFile(path string, file *fs_pb.File) error {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	bytes, err := file.Marshal()
	if err != nil {
		return err
	}
	strBytes := util.Base64Encode(bytes)
	r := NewStoreRequest(dataOpStoreFile, path, strBytes)

	select {
	case s.fileReqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.fileReqC <- r:
		case <-timer.C:
			return ErrWriteTimeout
		}
	}

	res := r.result()
	return res.err
}

func (s *Storage) StoreBatch(files map[string]*control_pb.FileInfo) error {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	batch := make(map[string]string, len(files))
	for path, file := range files {

		bytes, err := file.Marshal()
		if err != nil {
			return err
		}
		strBytes := util.Base64Encode(bytes)

		batch[path] = strBytes
	}

	r := NewStoreRequest(dataOpStoreBatch, batch)

	select {
	case s.fileReqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.fileReqC <- r:
		case <-timer.C:
			return ErrWriteTimeout
		}
	}

	res := r.result()
	return res.err
}

func (s *Storage) GetFile(path string) (*fs_pb.File, error) {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	r := NewStoreRequest(dataOpGetFile, path)
	select {
	case s.fileReqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.fileReqC <- r:
		case <-timer.C:
			return nil, ErrWriteTimeout
		}
	}

	res := r.result()
	bytes, err := util.Base64Decode(res.reply.(string))
	if err != nil {
		return nil, err
	}

	var file fs_pb.File
	err = file.Unmarshal(bytes)
	if err != nil {
		return nil, err
	}

	return &file, res.err
}

func (s *Storage) DeleteFile(path string) error {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	r := NewStoreRequest(dataOpDeleteFile, path)
	select {
	case s.fileReqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.fileReqC <- r:
		case <-timer.C:
			return ErrWriteTimeout
		}
	}

	res := r.result()
	return res.err
}

/////////////////////////////
// --------------------------
// Blocks methods
// --------------------------

func (s *Storage) PutBlocks(fileID string, blocks []*control_pb.BlockInfo) error {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	fBlocks := &control_pb.FileBlocks{
		Blocks: blocks,
	}
	bytes, _ := fBlocks.Marshal()
	strBytes := util.Base64Encode(bytes)

	r := NewStoreRequest(dataOpStoreBlocks, fileID, strBytes)

	select {
	case s.fileReqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.blocksReqC <- r:
		case <-timer.C:
			return ErrWriteTimeout
		}
	}

	res := r.result()
	return res.err
}

func (s *Storage) PutBatchBlocks(blocks map[string][]*control_pb.BlockInfo) error {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	data := make(map[string]string)

	for fileID, b := range blocks {
		fBlocks := &control_pb.FileBlocks{
			Blocks: b,
		}
		bytes, _ := fBlocks.Marshal()
		strBytes := util.Base64Encode(bytes)

		data[fileID] = strBytes
	}

	r := NewStoreRequest(dataOpStoreBatchBlocks, data)

	select {
	case s.blocksReqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.fileReqC <- r:
		case <-timer.C:
			return ErrWriteTimeout
		}
	}

	res := r.result()
	return res.err
}

func (s *Storage) GetBlocks(fileID []string) ([]*control_pb.BlockInfo, error) {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	r := NewStoreRequest(dataOpGetBlocks, fileID)

	select {
	case s.blocksReqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.fileReqC <- r:
		case <-timer.C:
			return nil, ErrWriteTimeout
		}
	}

	res := r.result()
	if res.err != nil {
		return nil, res.err
	}

	bytes, err := util.Base64Decode(res.reply.(string))
	if err != nil {
		return nil, err
	}

	var fBlocks control_pb.FileBlocks
	err = fBlocks.Unmarshal(bytes)
	if err != nil {
		return nil, err
	}

	return fBlocks.Blocks, nil
}

func (s *Storage) RemoveBlocks(fileID string) error {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	r := NewStoreRequest(dataOpDeleteBlocks, fileID)

	select {
	case s.blocksReqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.fileReqC <- r:
		case <-timer.C:
			return ErrWriteTimeout
		}
	}

	res := r.result()
	return res.err
}

////////////////////////////
// -------------------------
// Other logic
// -------------------------

func (s *Storage) NotifyShutdown() <-chan struct{} {
	return s.shutdownC
}

func (s *Storage) Shutdown() {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return
	}
	s.mu.RUnlock()

	close(s.shutdownC)

	s.mu.Lock()
	s.shutdown = true
	s.mu.Unlock()
}

type StoreResp struct {
	reply interface{}
	err   error
}

type StoreReq struct {
	op   dataOp
	args []interface{}
	resp chan *StoreResp
}

func NewStoreRequest(op dataOp, args ...interface{}) StoreReq {
	return StoreReq{op: op, args: args, resp: make(chan *StoreResp, 1)}
}

func (dr *StoreReq) done(reply interface{}, err error) {
	if dr.resp == nil {
		return
	}
	dr.resp <- &StoreResp{reply: reply, err: err}
}

func (dr *StoreReq) result() *StoreResp {
	if dr.resp == nil {
		// No waiting, as caller didn't care about response.
		return &StoreResp{}
	}
	return <-dr.resp
}

type dataOp int

const (
	dataOpStoreFile dataOp = iota
	dataOpStoreBatch
	dataOpGetFile
	dataOpDeleteFile

	dataOpStoreBlocks
	dataOpStoreBatchBlocks
	dataOpGetBlocks
	dataOpDeleteBlocks
)
