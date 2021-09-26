package registry

type RegistryStorageRedis struct {
	Registry *Registry
}

func NewRedisRegistryStorage(registry *Registry) RegistryStorage {
	return &RegistryStorageRedis{
		Registry: registry,
	}
}

func (r *RegistryStorageRedis) Run() error {

	go func() {
		for {
			select {
			case <-r.Registry.NotifyShutdown():
				return
			}
		}
	}()

	return nil
}

func (s *RegistryStorageRedis) RegisterNode(key string, data map[string]interface{}) error {

	return nil
}

func (s *RegistryStorageRedis) RemoveNode(key string) error {

	return nil
}

func (s *RegistryStorageRedis) GetNodes() map[string]map[string]interface{} {
	return nil
}
