package registry

import (
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	"sync"
	"time"
)

type RegistriesMap struct {
	mu         sync.RWMutex
	currentUID string
	registries map[string]*registry_pb.RegistryInfo
	updates    map[string]int64

	cleaningInterval time.Duration
}

func NewRegistriesMap() *RegistriesMap {
	return &RegistriesMap{

		registries: make(map[string]*registry_pb.RegistryInfo),
		updates:    make(map[string]int64),

		cleaningInterval: time.Second * 5,
	}
}

func (m *RegistriesMap) List() map[string]*registry_pb.RegistryInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nodes := make(map[string]*registry_pb.RegistryInfo, len(m.registries))
	for uid, info := range m.registries {
		nodes[uid] = info
	}

	return nodes
}

func (m *RegistriesMap) Add(name string, info *registry_pb.RegistryInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.registries[name]; !ok {
		m.registries[name] = info
	}

	m.updates[name] = time.Now().Unix()

	return nil
}

func (m *RegistriesMap) Remove(name string) error {
	m.mu.Lock()
	delete(m.registries, name)
	delete(m.updates, name)
	m.mu.Unlock()

	return nil
}

func (m *RegistriesMap) RegistryExists(uid string) bool {
	m.mu.RLock()
	_, ok := m.registries[uid]
	m.mu.RUnlock()

	return ok
}

func (m *RegistriesMap) RunCleaning() {

	ticker := time.NewTicker(m.cleaningInterval)
	for {
		select {
		case <-ticker.C:
			m.clean(time.Minute)
		}
	}
}

func (m *RegistriesMap) clean(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name := range m.registries {
		if name == m.currentUID {
			// no need to clean info for current registry instance
			continue
		}
		lastUpdated, ok := m.updates[name]
		if !ok {
			// As we do all operations with nodes under lock this should never happen.
			delete(m.registries, name)
		}

		if time.Now().Unix()-lastUpdated > int64(delay.Seconds()) {
			delete(m.registries, name)
			delete(m.updates, name)
		}
	}

}
