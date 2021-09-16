package storage

import (
	rhosus_node "parasource/rhosus/rhosus/node"
	"parasource/rhosus/rhosus/rlog"
	"parasource/rhosus/rhosus/storage/finder"
	"parasource/rhosus/rhosus/storage/storage_object"
	"path"
	"strconv"
	"sync"
)

type Volume struct {
	Node       *rhosus_node.Node
	Id         uint64
	Dir        string
	Collection string
	Finder     finder.Finder

	dataMu        sync.RWMutex
	MemoryMaxSize uint32

	reqCh    chan *so.StorageObjectRequest
	Location *DiskLocation

	lastIoError  error
	isCompacting bool
}

func NewVolume(dirname string, collection string, id uint64, preallocate int64, memoryMaxSize uint32) (*Volume, error) {
	v := &Volume{
		Id:            id,
		Dir:           dirname,
		Collection:    collection,
		MemoryMaxSize: memoryMaxSize,
		reqCh:         make(chan *so.StorageObjectRequest, 128),
	}
	// todo

	return v, nil
}

func VolumeFileName(dir string, collection string, id int) string {
	idString := strconv.Itoa(id)
	if collection == "" {
		return path.Join(dir, idString)
	}

	return path.Join(dir, collection+"_"+idString)
}

func (v *Volume) DataFileName() string {
	return VolumeFileName(v.Dir, v.Collection, int(v.Id))
}

func (v *Volume) ContentSize() uint64 {
	v.dataMu.RLock()
	defer v.dataMu.RUnlock()

	if v.Finder == nil {
		return 0
	}

	return v.Finder.ContentSize()
}

func (v *Volume) FilesCount() uint64 {
	v.dataMu.RLock()
	defer v.dataMu.RUnlock()

	if v.Finder == nil {
		return 0
	}
	return uint64(v.Finder.FileCount())
}

func (v *Volume) DeletedCount() uint64 {
	v.dataMu.RLock()
	defer v.dataMu.RUnlock()
	if v.Finder == nil {
		return 0
	}
	return uint64(v.Finder.DeletedCount())
}

func (v *Volume) MaxFileKey() string {
	v.dataMu.RLock()
	defer v.dataMu.RUnlock()
	if v.Finder == nil {
		return ""
	}
	return v.Finder.MaxFileKey()
}

func (v *Volume) Close() {
	v.dataMu.Lock()
	defer v.dataMu.Unlock()

	if v.Finder != nil {
		if err := v.Finder.Sync(); err != nil {
			v.Node.Logger.Log(rlog.NewLogEntry(rlog.LogLevelError, "error syncing volume before shutdown", map[string]interface{}{"vid": v.Id, "error": err}))
		}
		v.Finder.Close()
		v.Finder = nil
	}

	// Something related to DataBackend

}
