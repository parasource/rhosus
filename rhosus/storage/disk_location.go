package storage

import (
	"errors"
	"github.com/parasource/rhosus/rhosus/node"
	"github.com/parasource/rhosus/rhosus/storage/stats"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"sync"
	"time"
)

type DiskLocation struct {
	Node *rhosus_node.Node

	Id             string
	Directory      string
	MaxVolumeCount int

	volumesMu sync.RWMutex
	volumes   map[string]*Volume

	MinFreeSpace MinFreeSpace

	outOfSpace bool
}

func NewDiskLocation(dir string, maxVolumeCount int, minFreeSpace MinFreeSpace) (*DiskLocation, error) {
	dir = util.ResolvePath(dir)

	location := &DiskLocation{
		Directory:      dir,
		MaxVolumeCount: maxVolumeCount,
		MinFreeSpace:   minFreeSpace,
	}
	location.volumes = make(map[string]*Volume)
	go location.checkFreeSpace()

	return location, nil
}

func (l *DiskLocation) LoadVolume(vid string, kind int) error {
	return nil
}

func (l *DiskLocation) DeleteVolume(vid string) error {
	l.volumesMu.Lock()
	defer l.volumesMu.Unlock()

	_, ok := l.volumes[vid]
	if !ok {
		return errors.New("volume not found")
	}

	err := l.deleteVolumeById(vid)
	return err
}

func (l *DiskLocation) deleteVolumeById(vid string) error {
	_, ok := l.volumes[vid]
	if !ok {
		return errors.New("volume not found")
	}
	//e = v.Destroy()
	//if e != nil {
	//	return
	//}
	delete(l.volumes, vid)
	return nil
}

func (l *DiskLocation) FindVolume(vid string) (*Volume, bool) {
	l.volumesMu.RLock()
	defer l.volumesMu.RUnlock()

	v, ok := l.volumes[vid]
	return v, ok
}

func (l *DiskLocation) VolumesLen() int {
	l.volumesMu.RLock()
	defer l.volumesMu.RUnlock()

	return len(l.volumes)
}

func (l *DiskLocation) checkFreeSpace() {
	ticker := tickers.SetTicker(time.Minute)
	defer tickers.ReleaseTicker(ticker)

	for {
		select {
		case <-ticker.C:
			if dir, err := filepath.Abs(l.Directory); err == nil {
				s, err := stats.GetDiskStats(dir)
				if err != nil {
					logrus.Errorf("error getting disk stats: %v", err)
				}

				isLow := l.MinFreeSpace.IsLow(s.Free, s.PercentFree)
				l.outOfSpace = isLow

				if isLow {
					logrus.Errorf("disk ran out of space: %v", err)
				}
			}
		case <-l.Node.NotifyShutdown():
			return
		}
	}
}
