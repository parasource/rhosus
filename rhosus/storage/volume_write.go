package storage

import (
	"errors"
	so "github.com/parasource/rhosus/rhosus/storage/storage_object"
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
	for _, fileKey := range v.files {
		removeVolumeFiles(v.DataFileName(fileKey))
	}

	return nil
}

func removeVolumeFiles(filename string) {
	os.Remove(filename + ".dat")
}

func (v *Volume) enqueueRequest(req *so.StorageObjectRequest) {
	v.reqCh <- req
}

func (v *Volume) isModified(o *so.StorageObject) bool {
	return false
}

func (v *Volume) doWriteReq(o *so.StorageObject) (uint64, uint64, bool, error) {
	if !v.isModified(o) {
		return 0, o.DataSize, false, nil
	}

	// todo

	return 0, 0, false, nil
}

func (v *Volume) syncDelete(o *so.StorageObject) (uint64, error) {
	// todo

	return 0, nil
}

func (v *Volume) doDeleteReq(o *so.StorageObject) (uint64, error) {
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

			reqs := make([]*so.StorageObjectRequest, 0, 128)
			bytesToWrite := int64(0)
			//

			for {
				req, ok := <-v.reqCh
				if !ok {

				}

				if v.ContentSize()+uint64(bytesToWrite) > MaxVolumeSize {
					// По идее надо что-то делать, но пока не знаю что
				}

				reqs = append(reqs, req)
				bytesToWrite += int64(req.Size)

				if bytesToWrite >= 4*1024*1024 || len(reqs) >= 128 || len(v.reqCh) == 0 {
					break
				}
				if len(reqs) == 0 {
					continue
				}

				func() {
					v.dataMu.Lock()
					defer v.dataMu.Unlock()

					for i := 0; i < len(reqs); i++ {
						// todo check if write request

						offset, size, isModified, err := v.doWriteReq(reqs[i].Object)
						if err != nil {

						}

						reqs[i].Complete(offset, size, isModified, err)
					}
				}()
			}
		}
	}()
}

func (v *Volume) WriteBlob(id string, data []byte, size uint64) error {
	v.dataMu.Lock()
	defer v.dataMu.Unlock()

	return nil
}
