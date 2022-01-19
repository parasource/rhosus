package wal

type Snapshotter struct {
	dir       string
	parentDir string
}

type Snapshot struct {
	filename string
	size     uint64
}

func (s *Snapshotter) CreateSnapshot() (uint64, error) {
	return 0, nil
}
