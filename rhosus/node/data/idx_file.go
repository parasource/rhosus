package data

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/parasource/rhosus/rhosus/util/fileutil"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var (
	CorruptWriteErr = errors.New("idx file corrupted write")
)

const (
	idxBlockSize = 44
	idxFileSize  = partitionBlocksCount * idxBlockSize
)

type IdxBlock struct {
	ID     string
	Size   uint64
	Offset uint64
}

func (i IdxBlock) Marshal() ([]byte, error) {
	data := bytes.NewBuffer([]byte{})
	data.Write([]byte(i.ID))
	err := binary.Write(data, binary.BigEndian, i.Size)
	if err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}

func (i *IdxBlock) Unmarshal(data []byte) error {
	i.ID = string(data[:36])
	size := binary.BigEndian.Uint64(data[36:])
	i.Size = size
	return nil
}

type IdxFile struct {
	file *os.File

	lock sync.RWMutex
	len  int
}

func NewIdxFile(dir string, id string) (*IdxFile, error) {
	path := filepath.Join(dir, fmt.Sprintf("%v.idx", id))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		file.Close()
		os.Remove(path)
		return nil, err
	}
	err = fileutil.Preallocate(file, idxFileSize, true)
	if err != nil {
		file.Close()
		os.Remove(path)
		return nil, err
	}

	return &IdxFile{
		file: file,
		len:  0,
	}, nil
}

func (f *IdxFile) Load() map[string]IdxBlock {
	f.lock.Lock()
	defer f.lock.Unlock()

	var err error

	blocks := make(map[string]IdxBlock, partitionBlocksCount)
	offset := int64(0)
	for {
		data := make([]byte, idxBlockSize)
		_, err = f.file.ReadAt(data, offset)
		if err != nil {
			if err == io.EOF {
				break
			}
			offset += idxBlockSize
			continue
		}
		if !isBytesAllZeros(data) {
			var block IdxBlock
			err = block.Unmarshal(data)
			if err != nil {
				offset += idxBlockSize
				continue
			}

			block.Offset = uint64(offset)
			blocks[block.ID] = block
		}

		offset += idxBlockSize
	}

	return blocks
}

func (f *IdxFile) Write(blocks map[int]IdxBlock) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	for blockN, block := range blocks {
		data, err := block.Marshal()
		if err != nil {
			return err
		}
		n, err := f.file.WriteAt(data, int64(blockN*idxBlockSize))
		if err != nil {
			return err
		}
		if n != len(data) {
			return CorruptWriteErr
		}
	}
	return f.file.Sync()
}

func (f *IdxFile) Erase(blocks []int) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for _, blockN := range blocks {
		start := blockN * idxBlockSize
		end := (blockN + 1) * idxBlockSize
		for pos := start; pos < end; pos++ {
			_, err := f.file.WriteAt([]byte{0}, int64(pos))
			if err != nil {
				f.file.Sync()
				return
			}
		}
	}

	f.file.Sync()
}
