package registry

import (
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
)

type RegistryStorageRedis struct {
	shards *rhosus_redis.RedisShardPool
}

func NewRedisRegistryStorage(shardsPool *rhosus_redis.RedisShardPool) (*RegistryStorageRedis, error) {
	return &RegistryStorageRedis{
		shards: shardsPool,
	}, nil
}

func (r *RegistryStorageRedis) GetNodes() {

}
