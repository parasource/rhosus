package registry

import (
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"sort"
)

func blocksInfoToPlacement(blocks []*control_pb.BlockInfo) []*transport_pb.BlockPlacementInfo {
	placement := make([]*transport_pb.BlockPlacementInfo, len(blocks))
	//for i, block := range blocks {
	//	placement[i] = &transport_pb.BlockPlacementInfo{BlockID: block.Id, PartitionID: block.PartitionID}
	//}
	return placement
}

func fillAndSortBlocks(blocks []*control_pb.BlockInfo, dBlocks []*fs_pb.Block) []*fs_pb.Block {
	bMap := make(map[string]uint64, len(blocks))
	for _, block := range blocks {
		bMap[block.Id] = block.Index
	}
	for _, block := range dBlocks {
		block.Index = bMap[block.Id]
	}

	sort.Slice(dBlocks, func(i, j int) bool {
		return bMap[dBlocks[i].Id] < bMap[dBlocks[j].Id]
	})

	return dBlocks
}
