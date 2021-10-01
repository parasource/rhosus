package registry

import (
	"github.com/parasource/rhosus/rhosus/registry/redis"
)

type RegistryBrokerRedis struct {
	registry *Registry
	config   rhosus_redis.RedisConfig

	pool *rhosus_redis.RedisShardPool
}

func NewRegistryRedisBroker(registry *Registry, conf rhosus_redis.RedisConfig) (*RegistryBrokerRedis, error) {
	r := &RegistryBrokerRedis{
		registry: registry,
		config:   conf,
	}

	return r, nil
}

func (r *RegistryBrokerRedis) PublishCommand(pubReq rhosus_redis.PubRequest) error {

	//select {
	//case r.pool.
	//}

	return nil
}
