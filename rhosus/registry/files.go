package registry

import (
	"context"
	"errors"
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

var (
	ErrFileExists            = errors.New("file already exists")
	ErrNoSuchFileOrDirectory = errors.New("no such file or directory")
	ErrReplicationImpossible = errors.New("insufficient nodes to replicate to")
)

func (r *Registry) RegisterFile(file *control_pb.FileInfo) error {

	test, err := r.MemoryStorage.GetFileByPath(file.Path)
	if err != nil {
		return err
	}
	if test != nil {
		return ErrFileExists
	}

	sPath := strings.Split(file.Path, "/")
	if len(sPath) > 1 {
		parent, err := r.MemoryStorage.GetFileByPath(strings.Join(sPath[:len(sPath)-1], "/"))
		if err != nil {
			return err
		}
		if parent == nil {
			return ErrNoSuchFileOrDirectory
		}
		file.ParentID = parent.Id
	} else {
		file.ParentID = "root"
	}

	err = r.MemoryStorage.StoreFile(file)
	if err != nil {
		return err
	}

	return nil
}

func (r *Registry) TransportAndRegisterBlocks(fileID string, blocks []*fs_pb.Block, replicationFactor int) error {

	nodes := r.NodesManager.GetNodesWithLeastBlocks(replicationFactor)

	blockSeq := make(map[string]uint64, len(blocks))
	for _, block := range blocks {
		blockSeq[block.Id] = block.Index
	}

	// we can assign blocks to different nodes in parallel,
	// so we map blocks to nodes and process them separately
	blocksAssignment := make(map[*Node][]*fs_pb.Block, len(nodes))
	switch true {
	case len(nodes) < replicationFactor:
		// In this case we should disable replication for this file
		// and store all blocks on one node
		node := nodes[0]
		if _, ok := blocksAssignment[node]; !ok {
			blocksAssignment[node] = []*fs_pb.Block{}
		}

		blocksAssignment[node] = append(blocksAssignment[node], blocks...)
	case len(nodes) == replicationFactor:
		// In this case we just assign all blocks to all nodes
		for _, node := range nodes {
			blocksAssignment[node] = blocks
		}
	case len(nodes) > replicationFactor:
		// This is a bit trickier than other cases, since we have to choose
		// optimal node for every block
	}

	var resMu sync.Mutex
	assignResult := make(map[string][]*control_pb.BlockInfo_Placement, len(blocks))

	var wg sync.WaitGroup
	for node, blocks := range blocksAssignment {
		wg.Add(1)
		go func(node *Node, blocks []*fs_pb.Block) {
			defer wg.Done()

			start := time.Now()
			res, err := r.NodesManager.AssignBlocks(node.info.Id, blocks)
			if err != nil {
				logrus.Errorf("error assigning blocks to node: %v", err)
			}
			logrus.Infof("transported blocks to node %v in %v", node.info.Id, time.Since(start).String())

			resMu.Lock()
			for _, pInfo := range res {
				if _, ok := assignResult[pInfo.BlockID]; !ok {
					assignResult[pInfo.BlockID] = []*control_pb.BlockInfo_Placement{}
				}
				assignResult[pInfo.BlockID] = append(assignResult[pInfo.BlockID], &control_pb.BlockInfo_Placement{
					NodeID:      node.info.Id,
					PartitionID: pInfo.PartitionID,
				})
			}
			resMu.Unlock()
		}(node, blocks)
	}

	wg.Wait()

	// transfer is complete, now store blocks info
	var bInfos []*control_pb.BlockInfo
	for blockID, result := range assignResult {
		bInfos = append(bInfos, &control_pb.BlockInfo{
			Id:     blockID,
			Index:  blockSeq[blockID],
			FileID: fileID,
			Blocks: result,
		})
	}

	err := r.MemoryStorage.PutBlocks(bInfos)
	if err != nil {
		logrus.Errorf("error putting blocks: %v", err)
	}
	bs, _ := r.MemoryStorage.GetBlocks(fileID)
	logrus.Infof("TOTAL BLOCKS STORED: %v", len(bs))

	return nil
}

func (r *Registry) RemoveFileBlocks(file *control_pb.FileInfo) (error, map[string]error) {

	blocks, err := r.MemoryStorage.GetBlocks(file.Id)
	if err != nil {
		return fmt.Errorf("error getting blocks: %v", err), nil
	}

	bMap := make(map[string][]*transport_pb.BlockPlacementInfo)

	for _, block := range blocks {
		for _, placement := range block.Blocks {
			if _, ok := bMap[placement.NodeID]; !ok {
				bMap[placement.NodeID] = []*transport_pb.BlockPlacementInfo{}
			}
			bMap[placement.NodeID] = append(bMap[placement.NodeID], &transport_pb.BlockPlacementInfo{
				BlockID:     block.Id,
				PartitionID: placement.PartitionID,
			})
		}
	}

	var wg sync.WaitGroup

	for nodeId, blocks := range bMap {
		wg.Add(1)
		go func(nodeId string, blocks []*transport_pb.BlockPlacementInfo) {
			defer wg.Done()
			node := r.NodesManager.GetNode(nodeId)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			res, err := (*node.conn).RemoveBlocks(ctx, &transport_pb.RemoveBlocksRequest{
				Blocks: blocks,
			})
			if err != nil {
				logrus.Errorf("error removing blocks: %v", err)
				return
			}
			if !res.Success {
				logrus.Errorf("error removing blocks: %v", res.Error)
			}
		}(nodeId, blocks)
	}
	wg.Wait()

	err = r.MemoryStorage.DeleteFileWithBlocks(file)
	if err != nil {
		return err, nil
	}

	logrus.Infof("file deleted")

	return nil, nil
}

func (r *Registry) GetFileHandler(path string, transport func(block *fs_pb.Block)) error {

	file, err := r.MemoryStorage.GetFileByPath(path)
	if err != nil {
		return err
	}
	if file == nil {
		return ErrNoSuchFileOrDirectory
	}

	// Now we fetch BlockInfos from
	blocks, err := r.MemoryStorage.GetBlocks(file.Id)
	if err != nil {
		return fmt.Errorf("error getting blocks: %v", err)
	}

	// mapping blocks to nodes map
	bMap := make(map[string][]*transport_pb.BlockPlacementInfo)
	for _, block := range blocks {
		if _, ok := bMap[block.Blocks[0].NodeID]; !ok {
			bMap[block.Blocks[0].NodeID] = []*transport_pb.BlockPlacementInfo{}
		}

		bMap[block.Blocks[0].NodeID] = append(bMap[block.Blocks[0].NodeID], &transport_pb.BlockPlacementInfo{
			BlockID:     block.Id,
			PartitionID: block.Blocks[0].PartitionID,
		})
	}

	var result []*fs_pb.Block

	var wg sync.WaitGroup
	for nodeID, blocks := range bMap {
		wg.Add(1)
		go func(nodeID string, blocks []*transport_pb.BlockPlacementInfo) {
			defer wg.Done()
			actualBlocks, err := r.NodesManager.GetBlocks(nodeID, blocks)
			if err != nil {
				return
			}

			result = append(result, actualBlocks...)
		}(nodeID, blocks)
	}

	wg.Wait()

	result = fillAndSortBlocks(blocks, result)

	// todo: refactor
	for _, block := range result {
		transport(block)
	}

	return nil
}
