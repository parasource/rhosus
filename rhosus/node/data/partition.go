package data

import (
	"errors"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"sync"
	"time"
)

var (
	ErrWriteTimeout = errors.New("write timeout")
)

type sink struct {
	part *Partition

	blocks chan *fs_pb.Block
	lock   sync.Mutex
}

func newSink(part *Partition) *sink {
	return &sink{
		part:   part,
		blocks: make(chan *fs_pb.Block, 1000),
	}
}

func (s *sink) put(block *fs_pb.Block) error {
	select {
	case s.blocks <- block:
	default:
		ticker := tickers.SetTicker(time.Second)
		defer tickers.ReleaseTicker(ticker)
		select {
		case s.blocks <- block:
		case <-ticker.C:
			return ErrWriteTimeout
		}
	}

	return nil
}

func (s *sink) run() {
	ticker := tickers.SetTicker(time.Millisecond * 500)
	defer tickers.ReleaseTicker(ticker)

	for {
		select {
		case <-ticker.C:

			blocks := make(map[string][]byte, cap(s.blocks))

		loop:
			for {
				select {
				case block := <-s.blocks:
					blocks[block.Id] = block.Data
				default:
					break loop
				}
			}

			// no new blocks
			if len(blocks) == 0 {
				continue
			}

			if s.part.isAvailable(len(blocks)) {
				err, _ := s.part.writeBlocks(blocks)
				if err != nil {
					logrus.Errorf("error flushing partition sink: %v", err)
				}

				s.part.Sync()
			} else {
				// In this case we explicitly write blocks to first available partition
				// will work on this later
				// TODO

				count := s.part.getAvailableBlocks()
				smallerBlocks := make(map[string][]byte, count)
				i := 0
				for id, data := range blocks {
					if i == count {
						break
					}
					smallerBlocks[id] = data
					delete(blocks, id)
					i++
				}
				err, _ := s.part.writeBlocks(smallerBlocks)
				if err != nil {
					logrus.Errorf("error writing blocks to remaining space in partition: %v", err)
					continue
				}
				s.part.Sync()

				logrus.Infof("abandoning %v blocks", len(blocks))
				for range blocks {
					// TODO: write them to another partition and make an alias
				}

			}
		}
	}
}

type Partition struct {
	lock    sync.Mutex
	file    *os.File
	idxFile *IdxFile
	sink    *sink

	blocksMap map[string]IdxBlock

	// In some cases we might need to move blocks to another partition,
	// but registry will not know about that due to one directional connection.
	// So when the registry tries to find blocks that does not exist on the called partition,
	// it will look into moved blocks
	movedBlocks map[string]string

	ID           string
	checksumType control_pb.Partition_ChecksumType
	checksum     string
	full         bool
}

func newPartition(id string, file *os.File) (*Partition, error) {
	p := &Partition{
		file: file,

		blocksMap:    make(map[string]IdxBlock, partitionBlocksCount),
		movedBlocks:  make(map[string]string, partitionBlocksCount),
		ID:           id,
		checksumType: control_pb.Partition_CHECKSUM_CRC32,
		checksum:     "",
		full:         false,
	}
	idxFile, err := NewIdxFile("./parts", id)
	if err != nil {
		return nil, err
	}
	p.idxFile = idxFile
	p.sink = newSink(p)
	go p.sink.run()

	return p, nil
}

func (p *Partition) isAvailable(blocks int) bool {
	availableBlocks := partitionBlocksCount - len(p.blocksMap)

	return !p.full && availableBlocks >= blocks
}

func (p *Partition) getUsedBlocks() int {
	return len(p.blocksMap)
}

func (p *Partition) getAvailableBlocks() int {
	return partitionBlocksCount - len(p.blocksMap)
}

func (p *Partition) isBlockAllocated(n int) bool {
	for _, el := range p.blocksMap {
		if int(el.Offset/idxBlockSize) == n {
			return true
		}
	}
	return false
}

// writeBlocks writes blocks to partitions
func (p *Partition) writeBlocks(blocks map[string][]byte) (error, map[string]error) {

	errs := make(map[string]error, len(blocks))
	idxs := make(map[int]IdxBlock, len(blocks)) // header file for each block

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

		idxBlock := IdxBlock{
			ID:     id,
			Size:   uint64(len(data)),
			Offset: uint64(blockN * idxBlockSize),
		}
		p.blocksMap[id] = idxBlock
		if len(p.blocksMap) >= partitionBlocksCount {
			p.full = true
		}
		idxs[blockN] = idxBlock
	}

	err := p.idxFile.Write(idxs)
	if err != nil {
		return err, errs
	}

	//err = p.file.Sync()
	//if err != nil {
	//
	//}

	return nil, errs
}

func (p *Partition) Sync() error {
	return p.file.Sync()
}

func (p *Partition) loadHeader() error {

	data := make([]byte, idxFileSize)
	n, err := p.file.Read(data)
	if err != nil {
		return err
	}
	if n != idxFileSize {
		return ErrCorruptWrite
	}

	for _, block := range p.idxFile.Load() {
		p.blocksMap[block.ID] = block
	}

	return nil
}

func (p *Partition) writeBlockContents(block int, data []byte) (int, error) {
	offset := int64(block * defaultBlockSize)

	n, err := p.file.WriteAt(data, offset)
	if err != nil {
		logrus.Errorf("error writing block to file: %v", err)
		return 0, err
	}

	err = p.file.Sync()

	return n, err
}

func (p *Partition) readBlocks(blocks []*transport_pb.BlockPlacementInfo) map[string]*fs_pb.Block {
	result := make(map[string]*fs_pb.Block, len(blocks))

	for _, block := range blocks {
		var err error
		blockHeader := p.blocksMap[block.BlockID]
		blockN := blockHeader.Offset / idxBlockSize

		_, data, err := p.readBlockContents(int(blockN), blockHeader.Size)
		if err != nil {
			continue
		}

		result[block.BlockID] = &fs_pb.Block{
			Id:       block.BlockID,
			Index:    0,
			Offset:   0,
			Len:      blockHeader.Size,
			Data:     data,
			Checksum: nil,
		}
	}

	return result
}

func (p *Partition) readBlockContents(blockN int, size uint64) (int, []byte, error) {
	offset := int64(blockN * defaultBlockSize)

	data := make([]byte, size)
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
