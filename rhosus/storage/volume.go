package storage

import (
	rhosusnode "github.com/parasource/rhosus/rhosus/node"
	"github.com/parasource/rhosus/rhosus/storage/storage_object"
	"path"
	"sync"
)

const (
	MaxVolumeSize uint64 = 4 * 1024 * 1024 * 1024 * 8 // 32GB
)

type Volume struct {
	Node *rhosusnode.Node
	Id   uint64
	Dir  string

	dataMu        sync.RWMutex
	MemoryMaxSize uint32

	reqCh    chan *so.StorageObjectRequest
	Location *DiskLocation

	files []string

	lastIoError  error
	isCompacting bool
}

func NewVolume(dirname string, collection string, id uint64, preallocate int64, memoryMaxSize uint32) (*Volume, error) {
	v := &Volume{
		Id:            id,
		Dir:           dirname,
		MemoryMaxSize: memoryMaxSize,
		reqCh:         make(chan *so.StorageObjectRequest, 128),
	}
	// todo

	return v, nil
}

func VolumeFileName(dir string, key string) string {
	return path.Join(dir, key)
}

func (v *Volume) DataFileName(fileKey string) string {
	return VolumeFileName(v.Dir, fileKey)
}

func (v *Volume) ContentSize() uint64 {
	v.dataMu.RLock()
	defer v.dataMu.RUnlock()

	// todo
	return 0
}

func (v *Volume) FilesCount() uint64 {
	v.dataMu.RLock()
	defer v.dataMu.RUnlock()

	// todo
	return 0
}

func (v *Volume) DeletedCount() uint64 {
	v.dataMu.RLock()
	defer v.dataMu.RUnlock()

	// todo
	return 0
}

func (v *Volume) MaxFileKey() string {
	v.dataMu.RLock()
	defer v.dataMu.RUnlock()

	// todo
	return ""
}

func (v *Volume) Close() {
	v.dataMu.Lock()
	defer v.dataMu.Unlock()

	// todo

}
