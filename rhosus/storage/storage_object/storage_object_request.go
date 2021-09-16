package so

type StorageObjectRequest struct {
	Object     *StorageObject
	ActualSize int64
	Offset     uint64
	Size       uint64
	doneCh     chan interface{}
	modified   bool
	err        error
}

func NewStorageObjectRequest(o *StorageObject) *StorageObjectRequest {
	return &StorageObjectRequest{
		Object:     o,
		ActualSize: 0,
		Offset:     0,
		Size:       0,
		doneCh:     make(chan interface{}),
		modified:   true,
		err:        nil,
	}
}

func (s *StorageObjectRequest) WaitComplete() (uint64, uint64, bool, error) {
	<-s.doneCh
	return s.Offset, s.Size, s.modified, s.err
}

func (s *StorageObjectRequest) Complete(offset uint64, size uint64, modified bool, err error) {
	s.Offset = offset
	s.Size = size
	s.modified = modified
	s.err = err
	close(s.doneCh)
}

func (s *StorageObjectRequest) Submit() {
	close(s.doneCh)
}

func (s *StorageObjectRequest) IsSucceed() bool {
	return s.err == nil
}
