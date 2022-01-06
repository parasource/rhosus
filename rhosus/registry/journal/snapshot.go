package journal

type FileSnapshotStore struct {

}

type FileSnapshotSink struct {
	store *FileSnapshotStore
	dir string
	parentDir string
}

type Snapshot struct {

}

func NewFileSnapshotSink() (*FileSnapshotSink, error) {
	return &FileSnapshotSink{}, nil
}

func (f *FileSnapshotSink) CreateSnapshot() {

}