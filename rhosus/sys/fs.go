package sys

import (
	"github.com/parasource/rhosus/rhosus/util"
	"sync"
)

type FSConfig struct {
	BlockSizeLimit int64
	CacheDir       string
	CacheSizeLimit int64

	EncryptionEnabled bool
}

type FS struct {
	config FSConfig

	filesMu sync.Mutex
	files   map[uint64]*File

	bufPool sync.Pool

	signature int32
}

func NewRhosusFS(conf FSConfig) (*FS, error) {
	fs := &FS{
		config: conf,
		files:  make(map[uint64]*File),
		bufPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, conf.BlockSizeLimit)
			},
		},
		signature: util.RandomInt32(),
	}

	return fs, nil
}
