package registry

import (
	"errors"
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
)

// RegistryStorageRedis ///////////////////////
// Registry Storage that stores mappings files and bytes in Redis database
type RegistryStorageRedis struct {
	Registry *Registry

	shards []*rhosus_redis.RedisShard
}

func NewRedisRegistryStorage(registry *Registry, shardsConf []rhosus_redis.RedisShardConfig) (*RegistryStorageRedis, error) {

	var shards []*rhosus_redis.RedisShard

	if shardsConf == nil || len(shardsConf) == 0 {
		return nil, errors.New("redis shards are not specified. either use different driver, or specify redis shards")
	}

	for _, conf := range shardsConf {
		shard, err := rhosus_redis.NewShard(registry, conf)
		if err != nil {
			return nil, err
		}
		shards = append(shards, shard)
	}

	return &RegistryStorageRedis{
		Registry: registry,
		shards:   shards,
	}, nil
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
