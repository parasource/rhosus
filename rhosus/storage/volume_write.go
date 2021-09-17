package storage

import (
	"errors"
	"github.com/parasource/rhosus/rhosus/storage/finder"
	"github.com/parasource/rhosus/rhosus/storage/storage_object"
	"os"
)

var (
	ErrorNotFound         = errors.New("not found")
	ErrorDeleted          = errors.New("already deleted")
	ErrorSizeMismatch     = errors.New("error size mismatch")
	ErrorVolumeCompacting = errors.New("volume is compacting")
)

func (v *Volume) checkReadWriteError(err error) {
	if err == nil {
		if v.lastIoError != nil {
			v.lastIoError = nil
		}
		return
	}
	if err.Error() == "input/output error" {
		v.lastIoError = err
	}
}

//func (v *Volume) isFileModified(o *storage_object.StorageObject) bool {
//
//}

func (v *Volume) Destroy() error {
	if v.isCompacting {
		return ErrorVolumeCompacting
	}
	close(v.reqCh)

	// todo

	v.Close()
	removeVolumeFiles(v.DataFileName())

	return nil
}

func removeVolumeFiles(filename string) {
	os.Remove(filename + ".dat")
	os.Remove(filename + ".idx")
	os.Remove(filename + ".vif")
	// sorted index file
	os.Remove(filename + ".sdx")
	// compaction
	os.Remove(filename + ".cpd")
	os.Remove(filename + ".cpx")
	// level db indx file
	os.RemoveAll(filename + ".ldb")
	// marker for damaged or incomplete volume
	os.Remove(filename + ".note")
}

func (v *Volume) enqueueRequest(req *so.StorageObjectRequest) {
	v.reqCh <- req
}

func (v *Volume) isModified(o *so.StorageObject) bool {
	return false
}

func (v *Volume) doWriteReq(o *so.StorageObject) (uint64, finder.Size, bool, error) {
	if !v.isModified(o) {
		size := finder.Size(o.DataSize)
		return 0, size, false, nil
	}

	// todo

	return 0, 0, false, nil
}

func (v *Volume) syncDelete(o *so.StorageObject) (finder.Size, error) {
	// todo

	return 0, nil
}

func (v *Volume) doDeleteReq(o *so.StorageObject) (finder.Size, error) {
	// todo

	return 0, nil
}

func (v *Volume) startWorker() {
	go func() {
		closedCh := false
		for {
			if closedCh {
				break
			}

			//reqs := make([]*storage_object.StorageObjectRequest, 0, 128)
			//bytesToWrite := int64(0)
			//
			//for {
			//	req, ok := <- v.reqCh
			//	if !ok {
			//		closedCh = true
			//		break
			//	}
			//
			//
			//}
		}
	}()
}

func (v *Volume) WriteBlob(id string, data []byte, size finder.Size) error {
	v.dataMu.Lock()
	defer v.dataMu.Unlock()

	return nil
}
