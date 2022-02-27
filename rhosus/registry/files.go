package registry

import (
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"github.com/sirupsen/logrus"
	"time"
)

func (r *Registry) RegisterFile(file *control_pb.FileInfo, data []byte) {

}

func (r *Registry) registerFileWithBlocks(file *control_pb.FileInfo, blocks []*fs_pb.Block) error {

	var id string
	for nid := range r.NodesManager.nodes {
		id = nid
	}

	start := time.Now()
	res, err := r.NodesManager.AssignBlocks(id, blocks)
	if err != nil {
		logrus.Errorf("error assigning blocks to node: %v", err)
	}

	err = r.MemoryStorage.StoreFile(file)
	if err != nil {
		logrus.Errorf("error storing file: %v", err)
	}

	logrus.Infof("received placement res: %v", res)
	logrus.Infof("time passed: %v", time.Since(start).String())

	var bInfos []*control_pb.BlockInfo
	for _, block := range res {
		bInfos = append(bInfos, &control_pb.BlockInfo{
			Id:          block.BlockID,
			Index:       0,
			FileID:      file.Id,
			NodeID:      id,
			PartitionID: block.PartitionID,
		})
	}
	err = r.MemoryStorage.PutBlocks(bInfos)
	if err != nil {
		logrus.Errorf("error putting blocks: %v", err)
	}

	return nil
}
