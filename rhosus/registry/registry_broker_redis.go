package registry

import (
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
)

type RegistryBrokerRedis struct {
	config rhosus_redis.RedisConfig

	pool *rhosus_redis.RedisShardPool
}

func NewRegistryRedisBroker(shardsPool *rhosus_redis.RedisShardPool) (*RegistryBrokerRedis, error) {
	r := &RegistryBrokerRedis{
		pool: shardsPool,
	}

	return r, nil
}

func (r *RegistryBrokerRedis) PublishCommand(pubReq rhosus_redis.PubRequest) error {
	return r.pool.Shards[0].Publish(pubReq)
}
