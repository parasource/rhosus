package registry

type RegistryConfig struct {
	address string
}

type Registry struct {
	config RegistryConfig
}

func NewRegistry(config RegistryConfig) (*Registry, error) {
	cfg := &Registry{
		config: config,
	}

	return cfg, nil
}
