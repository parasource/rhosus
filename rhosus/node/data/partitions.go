package data

import (
	"bytes"
	"errors"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util/fileutil"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func init() {
	rand.Seed(time.Now().Unix())
}

var (
	ErrShutdown     = errors.New("parts map is shut down")
	ErrNotFound     = errors.New("error partition not found")
	ErrCorruptWrite = errors.New("partition corrupted writing")
)

const (
	partitionHeaderSize = 36 * 256

	defaultPartitionsDir      = "./parts"
	defaultMinPartitionsCount = 1
	defaultPartitionSize      = partitionHeaderSize + (1 * 1024 * 1000) // default partition size is header + 1gb
	defaultBlockSize          = 16 << 20                                // by default, block size is 16mb so one partition fits 256 blocks
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
		blocksMap:      make(map[string]int, 256),
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
		for n := 0; n < 256; n++ {
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
		if len(p.occupiedBlocks) >= 256 {
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
	headerSizeBytes := 36 * 256

	data := make([]byte, headerSizeBytes)
	n, err := p.file.Read(data)
	if err != nil {
		return err
	}
	if n != headerSizeBytes {
		return ErrCorruptWrite
	}

	headerSecSize := 36
	offset := 0
	for i := 0; i < 256; i++ {
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

type PartitionsMap struct {
	parts map[string]*Partition

	dir                string // directory where to store parts files
	minPartitionsCount int    // to minimize hotspots each node starts with fixed number of parts
	partitionSize      int64
}

func NewPartitionsMap(dir string, size uint64) (*PartitionsMap, error) {

	if dir == "" {
		dir = defaultPartitionsDir
	}

	path, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	p := &PartitionsMap{
		parts: make(map[string]*Partition),
		dir:   path,

		minPartitionsCount: defaultMinPartitionsCount,
		partitionSize:      defaultPartitionSize,
	}

	if err = os.MkdirAll(path, 0750); err != nil {
		return nil, err
	}
	err = p.loadPartitions()
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *PartitionsMap) getRandomPartition() (*Partition, error) {
	var availableParts []string

	for id, partition := range p.parts {
		if !partition.full && len(partition.occupiedBlocks) != 256 {
			availableParts = append(availableParts, id)
		}
	}

	// All existing partitions are full, so we create a new one
	if len(availableParts) == 0 {
		id, err := p.createPartition()
		if err != nil {
			return nil, err
		}
		availableParts = append(availableParts, id)
	}
	partID := availableParts[rand.Intn(len(availableParts))]

	return p.parts[partID], nil
}

func (p *PartitionsMap) createPartition() (string, error) {
	v4uuid, _ := uuid.NewV4()
	id := v4uuid.String()

	path := filepath.Join(p.dir, id)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
	if err != nil {
		return "", err
	}

	err = fileutil.Preallocate(file, p.partitionSize, true)
	if err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", err
	}

	err = file.Sync()
	if err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", err
	}

	p.parts[id] = newPartition(id, file)

	return id, nil
}

func (p *PartitionsMap) GetAvailablePartitions(blocks int) map[string]*Partition {
	parts := make(map[string]*Partition, len(p.parts))

	for id, part := range p.parts {
		availableBlocks := 256 - len(part.occupiedBlocks)
		if !part.full && availableBlocks >= blocks {
			parts[id] = part
		}
	}

	return parts
}

func (p *PartitionsMap) GetPartitionIDs() []string {

	parts := make([]string, len(p.parts))
	for id := range p.parts {
		parts = append(parts, id)
	}

	return parts
}

func (p *PartitionsMap) loadPartitions() error {
	var err error

	fis, err := ioutil.ReadDir(p.dir)
	for _, fi := range fis {
		name := fi.Name()
		if fi.IsDir() || len(name) != 36 {
			continue
		}

		file, err := os.OpenFile(filepath.Join(p.dir, name), os.O_RDWR, 0777)
		if err != nil {
			// closing all previously opened parts
			//for _, partition := range p.parts {
			//	err := partition.file.Close()
			//	if err != nil {
			//		logrus.Errorf("error closing partition file: %v", err)
			//	}
			//}
			return err
		}

		part := newPartition(name, file)
		err = part.loadHeader()
		if err != nil {
			logrus.Errorf("error loading headers: %v", err)
			file.Close()
			continue
		}
		p.parts[name] = part
	}
	if len(p.parts) == 0 {
		start := time.Now()
		for len(p.parts) < p.minPartitionsCount {
			v4uuid, _ := uuid.NewV4()
			name := v4uuid.String()

			path := filepath.Join(p.dir, name)
			file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
			if err != nil {
				logrus.Errorf("error creating partition file: %v. Trying again", err)
				continue
			}

			err = fileutil.Preallocate(file, p.partitionSize, true)
			if err != nil {
				logrus.Errorf("error preallocating file: %v", err)
			}

			err = file.Sync()
			if err != nil {
				logrus.Errorf("error sync: %v", err)
			}

			p.parts[name] = newPartition(name, file)
		}
		logrus.Infof("parts created in %v", time.Since(start).String())
	}

	return err
}

func (p *PartitionsMap) getPartition(id string) (*Partition, error) {

	if _, ok := p.parts[id]; !ok {
		return nil, ErrNotFound
	}

	return p.parts[id], nil
}

func (p *PartitionsMap) Shutdown() error {

	for _, partition := range p.parts {
		err := partition.file.Close()
		if err != nil {
			logrus.Errorf("error while closing partition file: %v", err)
		}
	}

	return nil
}
