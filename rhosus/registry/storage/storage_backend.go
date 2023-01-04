package storage

import control_pb "github.com/parasource/rhosus/rhosus/pb/control"

type StorageBackend interface {
	StoreFilesBatch(files map[string]*control_pb.FileInfo) error
	GetFile(path string) (*control_pb.FileInfo, error)
	GetFilesBatch(paths []string) ([]*control_pb.FileInfo, error)
	GetAllFiles() ([]*control_pb.FileInfo, error)

	GetAllBlocks() ([]*control_pb.BlockInfo, error)
	PutBlocksBatch(blocks map[string]*control_pb.BlockInfo) error
	GetBlocksBatch(blocks []string) (map[string]string, error)
	RemoveBlocksBatch(blockIDs []string) error
}
