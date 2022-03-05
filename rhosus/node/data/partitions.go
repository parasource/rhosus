package data

import (
	"bytes"
	"errors"
	"github.com/parasource/rhosus/rhosus/util/fileutil"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
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
	defaultBlockSize     = 2 << 20 // by default, block size is 16mb so one partition fits 256 blocks
	partitionHeaderSize  = 36 * (1 << 30 / defaultBlockSize)
	defaultPartitionSize = partitionHeaderSize + (1 << 30) // default partition size is header + 4gb
	partitionBlocksCount = (1 << 30) / defaultBlockSize

	defaultPartitionsDir      = "./parts"
	defaultMinPartitionsCount = 1
)

type PartitionsMap struct {
	parts map[string]*Partition
	//idxFiles map[string]*IdxFile

	dir                string // directory where to store parts files
	minPartitionsCount int    // to minimize hotspots each node starts with fixed number of parts
	watchIntervalMs    int    // interval to watch to create new partition
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
		//idxFiles: make(map[string]*IdxFile),
		dir: path,

		minPartitionsCount: defaultMinPartitionsCount,
		partitionSize:      defaultPartitionSize,
		watchIntervalMs:    1500,
	}

	if err = os.MkdirAll(path, 0750); err != nil {
		return nil, err
	}
	err = p.loadPartitions()
	if err != nil {
		return nil, err
	}

	go p.watchCreatePartitions()

	return p, nil
}

func (p *PartitionsMap) watchCreatePartitions() {
	ticker := tickers.SetTicker(time.Millisecond * time.Duration(p.watchIntervalMs))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			partsAvailable := 0
			for _, partition := range p.parts {
				// todo: define threshold for creating new partitions
				logrus.Infof("---- BLOCKS AVAILABLE: %v", partitionBlocksCount-len(partition.blocksMap))
				if !partition.full && (partitionBlocksCount-len(partition.blocksMap)) >= 150 {
					partsAvailable++
				}
			}

			if partsAvailable < 1 {
				_, err := p.createPartition()
				if err != nil {
					logrus.Errorf("error creating partition: %v", err)
				}
			}
		}
	}
}

func (p *PartitionsMap) getRandomPartition() (*Partition, error) {
	var availableParts []string

	for id, partition := range p.parts {
		if !partition.full && len(partition.occupiedBlocks) != partitionBlocksCount {
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

func (p *PartitionsMap) GetNotFullPartitions() []*Partition {
	var parts []*Partition

	for _, part := range p.parts {
		if part.isAvailable(1) {
			parts = append(parts, part)
		}
	}

	return parts
}

func (p *PartitionsMap) GetAvailablePartitions(blocks int) map[string]*Partition {
	parts := make(map[string]*Partition, len(p.parts))

	for id, part := range p.parts {
		if part.isAvailable(blocks) {
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
		logrus.Infof("loaded partition: %v", part.blocksMap)
	}
	if len(p.parts) == 0 {
		logrus.Infof("creating partitions")
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
		logrus.Infof("partitions created in %v", time.Since(start).String())
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
