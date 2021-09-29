package registry

type RegistryStorage interface {
	RegisterNode(key string, data map[string]interface{}) error
	RemoveNode(key string) error
	GetNodes() map[string]map[string]interface{}
}
