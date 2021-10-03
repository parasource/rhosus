package registry

import "github.com/parasource/rhosus/rhosus/storage"

func (r *Registry) RegisterFile(dir string, name string, owner string, group string, data []byte) (*storage.StoragePlacementInfo, error) {
	return &storage.StoragePlacementInfo{}, nil
}

func (r *Registry) DeleteFile(dir string, name string) error {
	return nil
}
