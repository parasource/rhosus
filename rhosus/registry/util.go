package registry

import (
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"sort"
)

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
