package data

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"os"
	"path/filepath"
	"sync"
)

type IdxFile struct {
	file *os.File

	lock        sync.RWMutex
	writeOffset int64
	len         int
}

func NewIdxFile(dir string, id string) (*IdxFile, error) {
	path := filepath.Join(dir, fmt.Sprintf("%v.idx", id))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		file.Close()
		return nil, err
	}

	return &IdxFile{
		file: file,
		len:  0,
	}, nil
}

//func (f *IdxFile) WriteBlock(block *fs_pb.Block) error {
//	bytes, err := block.Marshal()
//	if err != nil {
//		return err
//	}
//
//	f.file.WriteAt(, f.writeOffset)
//}

func (f *IdxFile) GetBlock(id string) (*fs_pb.Block, error) {
	return nil, nil
}

func (f *IdxFile) EraseBlock(id string) error {
	return nil
}
