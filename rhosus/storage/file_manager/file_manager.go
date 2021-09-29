package fm

import (
	rhosus_node "github.com/parasource/rhosus/rhosus/node"
	fm_pb "github.com/parasource/rhosus/rhosus/pb/file_manager"
	"github.com/parasource/rhosus/rhosus/util/queue"
	"os"
)

var (
	OS_UID = uint32(os.Getuid())
	OS_GID = uint32(os.Getgid())
)

type FileManager struct {
	node          *rhosus_node.Node
	deletionQueue queue.Queue
}

func NewFileManager(node *rhosus_node.Node) *FileManager {
	m := &FileManager{
		node:          node,
		deletionQueue: queue.New(),
	}

	return m
}

func (m *FileManager) CreatePackage(uid string, dir string, p *fm_pb.Package) error {
	return nil
}

func (m *FileManager) AppendPackage(uid string, chunks []*fm_pb.FileChunk) error {
	return nil
}

func (m *FileManager) DeletePackage(dir string, uid string, signatures []int32) error {
	return nil
}
