package storage

import (
	"context"
	"errors"
	rhosus_etcd "github.com/parasource/rhosus/rhosus/etcd"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/timers"
	"time"
)

var (
	ErrWriteTimeout = errors.New("etcd storage write timeout")
)

const (
	filesPrefix = "rhosus/data/files/"
)

type Config struct {
	WriteTimeoutS int
}

type Storage struct {
	config Config
	etcd   *rhosus_etcd.EtcdClient

	reqC chan StoreReq
}

func NewStorage(config Config, etcd *rhosus_etcd.EtcdClient) (*Storage, error) {
	s := &Storage{
		config: config,
		etcd:   etcd,

		reqC: make(chan StoreReq),
	}

	go s.runPipelines()

	return s, nil
}

func (s *Storage) runPipelines() {

	var srqs []StoreReq

	for {

		select {
		case r := <-s.reqC:
			srqs = append(srqs, r)
		}

	loop:
		for len(srqs) < 512 {
			select {
			case sr := <-s.reqC:
				srqs = append(srqs, sr)
			default:
				break loop
			}
		}

		for i := range srqs {
			switch srqs[i].op {
			case dataOpStore:
				res, err := s.etcd.Put(context.Background(), srqs[i].args[0], srqs[i].args[1])

				srqs[i].done(res, err)
			case dataOpGet:
				res, err := s.etcd.Get(context.Background(), srqs[i].args[0])

				srqs[i].done(string(res.Kvs[0].Value), err)
			case dataOpDelete:
				res, err := s.etcd.Delete(context.Background(), srqs[i].args[0])

				srqs[i].done(res, err)
			}
		}

		srqs = nil
	}
}

func (s *Storage) PutFile(path string, file *fs_pb.File) error {

	path = prefixedFilePath(path)

	bytes, err := file.Marshal()
	if err != nil {
		return err
	}
	strBytes := util.Base64Encode(bytes)
	r := NewStoreRequest(dataOpStore, path, strBytes)

	select {
	case s.reqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.reqC <- r:
		case <-timer.C:
			return ErrWriteTimeout
		}
	}

	res := r.result()
	return res.err
}

func (s *Storage) GetFile(path string) (*fs_pb.File, error) {
	path = prefixedFilePath(path)

	r := NewStoreRequest(dataOpGet, path)
	select {
	case s.reqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.reqC <- r:
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
	path = prefixedFilePath(path)

	r := NewStoreRequest(dataOpDelete, path)
	select {
	case s.reqC <- r:
	default:
		timer := timers.SetTimer(time.Second * time.Duration(s.config.WriteTimeoutS))
		defer timers.ReleaseTimer(timer)
		select {
		case s.reqC <- r:
		case <-timer.C:
			return ErrWriteTimeout
		}
	}

	res := r.result()
	return res.err
}

func prefixedFilePath(path string) string {
	return filesPrefix + path
}

type StoreResp struct {
	reply interface{}
	err   error
}

type StoreReq struct {
	op   dataOp
	args []string
	resp chan *StoreResp
}

func NewStoreRequest(op dataOp, args ...string) StoreReq {
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
	dataOpStore dataOp = iota
	dataOpGet
	dataOpDelete
)
