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
}

func NewRegistriesMap(registry *Registry) *RegistriesMap {
	return &RegistriesMap{
		registries: make(map[string]*registry_pb.RegistryInfo),
		updates:    make(map[string]int64),
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

func (m *RegistriesMap) Get(uid string) *registry_pb.RegistryInfo {
	m.mu.RLock()
	node, ok := m.registries[uid]
	if !ok {
		// todo
		// log something
		// this can't happen in normal conditions
	}

	return node
}

func (m *RegistriesMap) Add(uid string, info *registry_pb.RegistryInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.registries[uid]; !ok {
		m.registries[uid] = info
	}

	m.updates[uid] = time.Now().Unix()
}

func (m *RegistriesMap) Remove(uid string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.registries[uid]; ok {
		delete(m.registries, uid)
		delete(m.updates, uid)
	}
}

func (m *RegistriesMap) clean(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for uid := range m.registries {
		if uid == m.currentUID {
			// no need to clean info for current registry instance
			continue
		}
		lastUpdated, ok := m.updates[uid]
		if !ok {
			// As we do all operations with nodes under lock this should never happen.
			delete(m.registries, uid)
		}

		if time.Now().Unix()-lastUpdated > int64(delay.Seconds()) {
			delete(m.registries, uid)
			delete(m.updates, uid)
		}
	}
}
