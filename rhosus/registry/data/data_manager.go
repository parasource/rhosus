package data

import (
	"github.com/parasource/rhosus/rhosus/node/profiler"
	"github.com/parasource/rhosus/rhosus/pb/fs"
	"github.com/parasource/rhosus/rhosus/registry"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"sync"
	"time"
)

type DataManager struct {
	Registry *registry.Registry

	mu       sync.RWMutex
	pages    map[string]*fs.Page
	profiler *profiler.Profiler
}

func NewDataManager(registry *registry.Registry) *DataManager {
	return &DataManager{
		Registry: registry,

		pages: make(map[string]*fs.Page),
	}
}

func (m *DataManager) Setup() error {
	var err error

	return err
}

// CreatePage creates a new page and returns it's uid
func (m *DataManager) CreatePage() (string, error) {
	if !m.canCreateNewPage() {
		return "", errorOutOfSpace
	}

	uid, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	page := &fs.Page{
		Uid:          uid.String(),
		UsedSpace:    0,
		Blocks:       make(map[string]*fs.Block),
		ChecksumType: fs.ChecksumType_CHECKSUM_CRC32,
		Checksum:     "",
		UpdatedAt:    time.Now().Unix(),
	}

	m.mu.Lock()
	m.pages[uid.String()] = page
	m.mu.Unlock()

	return uid.String(), nil
}

func (m *DataManager) AssignBlocks(blocks map[string]*fs.Block) (err error) {

	pages := make(map[string]*fs.Page, len(m.pages))
	m.mu.RLock()
	for uid, page := range m.pages {
		pages[uid] = page
	}
	m.mu.RUnlock()

	found := false
	for _, page := range pages {

		if found {
			break
		}

		if page.HasFreeSpace() {
			found = true

			// As the method lies in another package, we should use defer for unlocking mu
			err = func() error {
				m.mu.Lock()
				defer m.mu.Unlock()

				err := page.AppendBlocks(blocks)
				return err
			}()
			if err != nil {
				return err
			}

		}
	}

	return err
}

func (m *DataManager) GetPage(uid string) *fs.Page {
	m.mu.RLock()
	page := m.pages[uid]
	m.mu.RUnlock()

	return page
}

func (m *DataManager) canCreateNewPage() bool {
	// todo
	return true
}
