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

			if s.part.isAvailable(len(blocks)) {
				logrus.Infof("we're here")
				err, _ := s.part.writeBlocks(blocks)
				if err != nil {
					logrus.Errorf("error flushing partition sink: %v", err)
				}

				s.part.Sync()
			} else {
				// In this case we explicitly write blocks to first available partition
				// will work on this later
				// TODO
			}
		}
	}
}

type Partition struct {
	file *os.File
	sink *sink

	blocksMap      map[string]int
	occupiedBlocks []int

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

func newPartition(id string, file *os.File) *Partition {
	p := &Partition{
		file: file,

		blocksMap:      make(map[string]int, partitionBlocksCount),
		movedBlocks:    make(map[string]string, partitionBlocksCount),
		occupiedBlocks: []int{},
		ID:             id,
		checksumType:   control_pb.Partition_CHECKSUM_CRC32,
		checksum:       "",
		full:           false,
	}
	p.sink = newSink(p)
	go p.sink.run()

	return p
}

func (p *Partition) isAvailable(blocks int) bool {
	availableBlocks := partitionBlocksCount - len(p.occupiedBlocks)

	return !p.full && availableBlocks >= blocks
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
