package backend

import (
	"errors"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/timers"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"sync"
	"time"
)

var (
	ErrWriteTimeout = errors.New("backend storage write timeout")
	ErrShutdown     = errors.New("backend storage is shut down")
)

const (
	defaultDbFilePath = "data.db"

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

func (s *Storage) StoreFilesBatch(files map[string]*control_pb.FileInfo) error {

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

	r := NewStoreRequest(dataOpStoreFiles, batch)

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

func (s *Storage) GetFile(path string) (*control_pb.FileInfo, error) {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	res, err := s.GetFilesBatch([]string{path})
	if err != nil {
		return nil, err
	}

	if len(res) < 1 {
		return nil, errors.New("wrong result length")
	}

	return res[0], nil
}

func (s *Storage) GetFilesBatch(paths []string) ([]*control_pb.FileInfo, error) {
	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	r := NewStoreRequest(dataOpGetFilesBatch, paths)
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
	if res.err != nil {
		return nil, res.err
	}

	var files []*control_pb.FileInfo

	for _, data := range res.reply.([]string) {

		bytes, err := util.Base64Decode(data)
		if err != nil {

		}

		var file control_pb.FileInfo
		err = file.Unmarshal(bytes)
		if err != nil {
			logrus.Errorf("error unmarshaling file info: %v", err)
		}

		files = append(files, &file)
	}

	return files, nil
}

func (s *Storage) GetAllFiles() ([]*control_pb.FileInfo, error) {
	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	r := NewStoreRequest(dataOpGetAllFiles)
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
	if res.err != nil {
		return nil, res.err
	}

	var files []*control_pb.FileInfo
	for _, data := range res.reply.([]string) {

		bytes, err := util.Base64Decode(data)
		if err != nil {

		}

		var file control_pb.FileInfo
		err = file.Unmarshal(bytes)
		if err != nil {
			logrus.Errorf("error unmarshaling file info: %v", err)
		}

		files = append(files, &file)
	}

	return files, nil
}

func (s *Storage) GetAllBlocks() ([]*control_pb.BlockInfo, error) {
	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	r := NewStoreRequest(dataOpGetAllBlocks)
	select {
	case s.blocksReqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.blocksReqC <- r:
		case <-timer.C:
			return nil, ErrWriteTimeout
		}
	}

	res := r.result()
	if res.err != nil {
		return nil, res.err
	}

	var blocks []*control_pb.BlockInfo
	for _, data := range res.reply.([]string) {

		bytes, err := util.Base64Decode(data)
		if err != nil {
			continue
		}

		var block control_pb.BlockInfo
		err = block.Unmarshal(bytes)
		if err != nil {
			logrus.Errorf("error unmarshaling file info: %v", err)
		}

		blocks = append(blocks, &block)
	}

	return blocks, nil
}

func (s *Storage) RemoveFilesBatch(fileIDs []string) error {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	r := NewStoreRequest(dataOpDeleteFiles, fileIDs)
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

// PutBlocksBatch is used by node to map block ids to partitions
func (s *Storage) PutBlocksBatch(blocks map[string]*control_pb.BlockInfo) error {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	data := make(map[string]string, len(blocks))
	for id, info := range blocks {
		bytes, err := info.Marshal()
		if err != nil {
			logrus.Errorf("error marshaling block info: %v", err)
			continue
		}
		data[id] = util.Base64Encode(bytes)
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

func (s *Storage) GetBlocksBatch(blocks []string) (map[string]string, error) {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return nil, ErrShutdown
	}
	s.mu.RUnlock()

	r := NewStoreRequest(dataOpGetBlocks, blocks)

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
	return res.reply.(map[string]string), res.err
}

func (s *Storage) RemoveBlocksBatch(blocks []string) error {

	s.mu.RLock()
	if s.shutdown {
		s.mu.RUnlock()
		return ErrShutdown
	}
	s.mu.RUnlock()

	r := NewStoreRequest(dataOpDeleteBlocks, blocks)

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
	dataOpStoreFiles dataOp = iota
	dataOpGetFilesBatch
	dataOpGetAllFiles
	dataOpDeleteFiles

	dataOpStoreBatchBlocks
	dataOpGetBlocks
	dataOpGetAllBlocks
	dataOpDeleteBlocks
)
