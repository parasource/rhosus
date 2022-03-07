package registry

import (
	"errors"
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

var (
	ErrFileExists            = errors.New("file already exists")
	ErrNoSuchFileOrDirectory = errors.New("no such file or directory")
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

	// just getting first node for example
	var nodeID string
	for id := range r.NodesManager.nodes {
		nodeID = id
		break
	}

	actualBlocks, err := r.NodesManager.GetBlocks(nodeID, blocksInfoToPlacement(blocks))
	if err != nil {
		return fmt.Errorf("error getting blocks from node: %v", err)
	}
	actualBlocks = fillAndSortBlocks(blocks, actualBlocks)

	// todo: refactor
	for _, block := range actualBlocks {
		transport(block)
	}

	return nil
}
