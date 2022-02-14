package registry

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/backend"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"testing"
)

var (
	dbPath = path.Join(os.TempDir(), "test.db")
)

const (
	filesCount = 10000
)

func mockRegistry(t *testing.T) *Registry {
	t.Helper()

	b, err := backend.NewStorage(backend.Config{
		DbFilePath:    dbPath,
		WriteTimeoutS: 1,
		NumWorkers:    1,
	})
	assert.Nil(t, err)

	return &Registry{
		Backend: b,
	}
}

func TestMemoryStorage_FlushFiles(t *testing.T) {
	r := mockRegistry(t)
	s, err := NewMemoryStorage(r)
	assert.Nil(t, err)

	for i := 1; i <= filesCount; i++ {
		err := s.StoreFile(&control_pb.FileInfo{
			Type:  control_pb.FileInfo_FILE,
			Id:    fmt.Sprintf("index_%v.html", i),
			Path:  fmt.Sprintf("Desktop/index_%v.html", i),
			Size_: 64,
			Owner: "eovchinnikov",
			Group: "admin",
		})
		assert.Nil(t, err)
	}

	txn := s.db.Txn(false)
	err = s.flushFilesToBackend(txn)
	assert.Nil(t, err)

	r.Backend.Shutdown()
	os.Remove(dbPath)
}
