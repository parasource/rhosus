package storage

import (
	"parasource/rhosus/rhosus/storage/data/finder"
	"sync"
)

type Volume struct {
	id     uint64
	dir    string
	finder finder.Finder

	filesMu sync.RWMutex
}
