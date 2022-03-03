package data

import (
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

type Partition struct {
	file *os.File

	blocksMap      map[string]int
	occupiedBlocks []int

	ID           string
	checksumType control_pb.Partition_ChecksumType
	checksum     string
	full         bool
}

func newPartition(id string, file *os.File) *Partition {
	return &Partition{
		file:           file,
		blocksMap:      make(map[string]int, partitionBlocksCount),
		occupiedBlocks: []int{},
		ID:             id,
		checksumType:   control_pb.Partition_CHECKSUM_CRC32,
		checksum:       "",
		full:           false,
	}
}

func (p *Partition) isBlockAllocated(n int) bool {
	for _, el := range p.occupiedBlocks {
		if el == n {
			return true
		}
	}
	return false
}

// writeBlocks writes blocks to partitions
func (p *Partition) writeBlocks(blocks map[string][]byte) (error, map[string]error) {

	errs := make(map[string]error, len(blocks))
	headerRecs := make(map[string]int, len(blocks)) // header file for each block

	for id, data := range blocks {
		// getting number of first available block
		var blockN int
		for n := 0; n < partitionBlocksCount; n++ {
			if !p.isBlockAllocated(n) {
				blockN = n
				break
			}
		}

		n, err := p.writeBlockContents(blockN, data)
		if err != nil {
			errs[id] = err
			continue
		}
		if len(data) != n {
			errs[id] = ErrCorruptWrite
			continue
		}

		p.occupiedBlocks = append(p.occupiedBlocks, blockN)
		p.blocksMap[id] = blockN
		if len(p.occupiedBlocks) >= partitionBlocksCount {
			p.full = true
		}
		headerRecs[id] = blockN
	}

	err := p.writeHeaderRecs(headerRecs)
	if err != nil {
		return err, errs
	}

	err = p.file.Sync()
	if err != nil {

	}

	return nil, errs
}

func (p *Partition) Sync() error {
	return p.file.Sync()
}

func (p *Partition) writeHeaderRecs(recs map[string]int) error {
	for id, n := range recs {
		n, err := p.file.WriteAt([]byte(id), int64(36*n))
		if err != nil || n != 36 {
			// todo: maybe retry?
		}
	}

	return nil
}

func (p *Partition) loadHeader() error {

	data := make([]byte, partitionHeaderSize)
	n, err := p.file.Read(data)
	if err != nil {
		return err
	}
	if n != partitionHeaderSize {
		return ErrCorruptWrite
	}

	headerSecSize := 36
	offset := 0
	for i := 0; i < partitionBlocksCount; i++ {
		id := data[offset : offset+headerSecSize]
		if !isIdAllZeros(id) {
			p.blocksMap[string(id)] = i
			p.occupiedBlocks = append(p.occupiedBlocks, i)
		}
		offset += headerSecSize
	}

	return nil
}

func (p *Partition) writeBlockContents(block int, data []byte) (int, error) {
	offset := int64(partitionHeaderSize + block*defaultBlockSize)

	n, err := p.file.WriteAt(data, offset)
	if err != nil {
		logrus.Errorf("error writing block to file: %v", err)
		return 0, err
	}

	err = p.file.Sync()

	return n, err
}

func (p *Partition) readBlocks(blocks []*transport_pb.BlockPlacementInfo) map[string][]byte {
	result := make(map[string][]byte, len(blocks))

	for _, block := range blocks {
		var err error
		offset := p.blocksMap[block.BlockID]
		_, result[block.BlockID], err = p.readBlockContents(offset)
		if err != nil {
			delete(result, block.BlockID)
			continue
		}
	}

	return result
}

func (p *Partition) readBlockContents(blockN int) (int, []byte, error) {
	offset := int64(partitionHeaderSize + blockN*defaultBlockSize)

	data := make([]byte, defaultBlockSize)
	n, err := p.file.ReadAt(data, offset)
	if err != nil {
		logrus.Errorf("error reading blockN from file: %v", err)
		return 0, nil, err
	}

	return n, data, err
}

func (p *Partition) GetFile() io.Reader {
	return p.file
}
