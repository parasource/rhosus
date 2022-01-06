package data

import (
	"github.com/parasource/rhosus/rhosus/node/profiler"
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"github.com/parasource/rhosus/rhosus/registry"
	"github.com/sirupsen/logrus"
	"sync"
)

type DataManager struct {
	Registry *registry.Registry

	mu       sync.RWMutex
	pages    map[string]*Page
	profiler *profiler.Profiler
}

func NewDataManager(registry *registry.Registry) *DataManager {
	return &DataManager{
		Registry: registry,

		pages: make(map[string]*Page),
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

	page, err := NewPage()
	if err != nil {
		logrus.Errorf("cannot create new page: %v", err)
	}

	m.mu.Lock()
	m.pages[page.Uid] = page
	m.mu.Unlock()

	return page.Uid, nil
}

func (m *DataManager) AssignBlocks(blocks map[string]*fs_pb.Block) (err error) {

	pages := make(map[string]*Page, len(m.pages))
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

func (m *DataManager) canCreateNewPage() bool {
	// todo
	return true
}
