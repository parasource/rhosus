package registry

import (
	"github.com/gomodule/redigo/redis"
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
)

type RegistryStorageRedis struct {
	Registry *Registry

	shards *rhosus_redis.RedisShardPool
}

func NewRedisRegistryStorage(registry *Registry, shardsConf []rhosus_redis.RedisShardConfig) (*RegistryStorageRedis, error) {

	shardsPool, err := rhosus_redis.NewRedisShardPool(registry, shardsConf)
	if err != nil {

	}
	shardsPool.SetMessagesHandler(func(message redis.Message) {
		switch message.Channel {
		case registryInfoChannel:
			var cmd *registry_pb.Command
			err := cmd.Unmarshal(message.Data)
			if err != nil {

			}

			var info *registry_pb.RegistryInfo
			err = info.Unmarshal(cmd.Data)
			if err != nil {

			}

			// todo
		}
	})
	return &RegistryStorageRedis{
		Registry: registry,
		shards:   shardsPool,
	}, nil
}

func (r *RegistryStorageRedis) GetNodes() {

}
