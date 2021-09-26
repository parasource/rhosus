package registry

type RegistryStorage interface {
	Run() error

	RegisterNode(key string, data map[string]interface{}) error
	RemoveNode(key string) error
	GetNodes() map[string]map[string]interface{}
}
