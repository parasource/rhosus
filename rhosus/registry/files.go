package registry

import (
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"github.com/sirupsen/logrus"
	"time"
)

func (r *Registry) RegisterFile(file *control_pb.FileInfo) error {
	err := r.MemoryStorage.StoreFile(file)
	if err != nil {
		return err
	}

	return nil
}

func (r *Registry) TransportAndRegisterBlocks(fileID string, blocks []*fs_pb.Block) error {

	var nodeID string
	for _, node := range r.NodesManager.nodes {
		nodeID = node.info.Id
		break
	}

	start := time.Now()
	res, err := r.NodesManager.AssignBlocks(nodeID, blocks)
	if err != nil {
		logrus.Errorf("error assigning blocks to node: %v", err)
	}
	logrus.Infof("transported blocks in %v", time.Since(start).String())

	bMap := make(map[string]*fs_pb.Block)
	for _, block := range blocks {
		bMap[block.Id] = block
	}

	var bInfos []*control_pb.BlockInfo
	for _, block := range res {
		bInfos = append(bInfos, &control_pb.BlockInfo{
			Id:          block.BlockID,
			Index:       bMap[block.BlockID].Index,
			FileID:      fileID,
			NodeID:      nodeID,
			PartitionID: block.PartitionID,
		})
		logrus.Infof("IDX STORED: %v", bMap[block.BlockID].Index)
	}

	err = r.MemoryStorage.PutBlocks(bInfos)
	if err != nil {
		logrus.Errorf("error putting blocks: %v", err)
	}
	bs, _ := r.MemoryStorage.GetBlocks(fileID)
	logrus.Infof("TOTAL BLOCKS STORED: %v", len(bs))

	return nil
}

func (r *Registry) GetFileHandler(path string) {

}
