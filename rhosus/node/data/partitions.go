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

var (
	ErrShutdown = errors.New("partitions map is shut down")
	ErrNotFound = errors.New("error partition not found")
)

const (
	defaultPartitionsDir      = "./parts"
	defaultMinPartitionsCount = 1
	defaultPartitionSizeMb    = 1 * 1000         // default partition size is 4gb
	defaultBlockSize          = 16 * 1000 * 1000 // by default, one partition fits 256 blocks
)

type Partition struct {
	ID           string
	checksumType control_pb.Partition_ChecksumType
	checksum     string

	file *os.File
}

func (p *Partition) getBlockContents(block int) (int, []byte, error) {
	offset := int64(block * defaultBlockSize)

	data := make([]byte, defaultBlockSize)
	n, err := p.file.ReadAt(data, offset)
	if err != nil {
		logrus.Errorf("error reading block from file: %v", err)
		return 0, nil, err
	}

	return n, data, err
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

func (p *Partition) GetFile() io.Reader {
	return p.file
}

type PartitionsMap struct {
	mu         sync.RWMutex
	partitions map[string]*Partition

	dir                string // directory where to store partitions files
	minPartitionsCount int    // to minimize hotspots each node starts with fixed number of partitions
	partitionSizeMb    int64

	shutdown         bool
	isReceivingPages bool
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
		partitions: make(map[string]*Partition),
		dir:        path,

		minPartitionsCount: defaultMinPartitionsCount,
		partitionSizeMb:    defaultPartitionSizeMb,
	}

	if err = os.MkdirAll(path, 0750); err != nil {
		return nil, err
	}
	err = p.loadPartitions()
	if err != nil {
		return nil, err
	}

	// testing
	for _, part := range p.partitions {
		var block []byte
		for i := 0; i < 100*1024*1000/2; i++ {
			block = append(block, byte('a'))
			block = append(block, byte('b'))
		}
		start := time.Now()
		n, err := part.writeBlockContents(0, block)
		if err != nil {
			logrus.Fatalf("error putting contents: %v", err)
		}
		logrus.Infof("put 100mb in %v", time.Since(start).String())

		n, block2, err := part.getBlockContents(0)
		if err != nil {
			logrus.Fatalf("error reading block: %v", err)
		}
		logrus.Info(n, block2[:16])
	}

	return p, nil
}

func (p *PartitionsMap) GetPartitionsIDs() []string {
	//p.mu.Lock()
	//defer p.mu.Unlock()

	parts := make([]string, len(p.partitions))
	for id := range p.partitions {
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
			// closing all previously opened partitions
			//for _, partition := range p.partitions {
			//	err := partition.file.Close()
			//	if err != nil {
			//		logrus.Errorf("error closing partition file: %v", err)
			//	}
			//}
			return err
		}

		p.partitions[name] = &Partition{
			ID:           name,
			checksumType: control_pb.Partition_CHECKSUM_CRC32,
			checksum:     "",
			file:         file,
		}
	}
	if len(p.partitions) == 0 {
		start := time.Now()
		for len(p.partitions) < p.minPartitionsCount {
			v4uuid, _ := uuid.NewV4()
			name := v4uuid.String()

			path := filepath.Join(p.dir, name)
			file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0666)
			if err != nil {
				logrus.Errorf("error creating partition file: %v. Trying again", err)
				continue
			}

			err = fileutil.Preallocate(file, p.partitionSizeMb*1024*1024, true)
			if err != nil {
				logrus.Errorf("error preallocating file: %v", err)
			}

			err = file.Sync()
			if err != nil {
				logrus.Errorf("error sync: %v", err)
			}

			p.partitions[name] = &Partition{
				ID:           name,
				checksumType: control_pb.Partition_CHECKSUM_CRC32,
				checksum:     "",
				file:         file,
			}
		}
		logrus.Infof("partitions created in %v", time.Since(start).String())
	}

	return err
}

func (p *PartitionsMap) GetPartition(id string) (*Partition, error) {

	//p.mu.RLock()
	//if p.shutdown {
	//	p.mu.RUnlock()
	//	return nil, ErrShutdown
	//}
	//p.mu.RUnlock()

	if _, ok := p.partitions[id]; !ok {
		return nil, ErrNotFound
	}

	return p.partitions[id], nil
}

func (p *PartitionsMap) Shutdown() error {

	p.mu.RLock()
	if p.shutdown {
		p.mu.RUnlock()
		return nil
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.shutdown = true
	for _, partition := range p.partitions {
		err := partition.file.Close()
		if err != nil {
			logrus.Errorf("error while closing partition file: %v", err)
		}
	}

	return nil
}
