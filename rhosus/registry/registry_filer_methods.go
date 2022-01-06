package registry

import (
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"github.com/parasource/rhosus/rhosus/sys"
	"github.com/parasource/rhosus/rhosus/util/uuid"
)

func (r *Registry) RegisterFile(dir string, name string, owner string, group string, timestamp int64, size uint64, data []byte) (*sys.File, error) {

	uid, _ := uuid.NewV4()

	info := fs_pb.File{
		Uid:       uid.String(),
		Name:      name,
		DirID:     "",
		FullPath:  "",
		Timestamp: timestamp,
		Size_:     size,
		Blocks:    int32(size / sys.BlockSizeMb),
	}
	_ = info.Blocks
	//err := r.Storage.RegisterFile(uid.String(), &info)
	//if err != nil {
	//	return nil, err
	//}

	return &sys.File{}, nil
}

func (r *Registry) DeleteFile(dir string, name string) error {

	return nil
}
