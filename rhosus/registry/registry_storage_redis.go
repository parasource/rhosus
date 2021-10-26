package registry

import (
	node_pb "github.com/parasource/rhosus/rhosus/pb/node"
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
)

type RegistryStorageRedis struct {
	registryId string
	shards     *rhosus_redis.RedisShardPool
}

func NewRedisRegistryStorage(shardsPool *rhosus_redis.RedisShardPool) (*RegistryStorageRedis, error) {
	return &RegistryStorageRedis{
		shards: shardsPool,
	}, nil
}

func (r *RegistryStorageRedis) GetNodes() {

}

func (r *RegistryStorageRedis) RegisterFile(id string, info *registry_pb.FileInfo) error {
	var err error

	shard := r.shards.GetShard(r.registryId)
	err = shard.RegisterFile(id, info)
	if err != nil {

	}

	return err
}

func (r *RegistryStorageRedis) RemoveFile(id string) error {
	var err error

	shard := r.shards.GetShard(r.registryId)
	err = shard.RemoveFile(id)
	if err != nil {

	}

	return err
}

func (r *RegistryStorageRedis) AddNode(uid string, info *node_pb.NodeInfo) error {
	var err error

	shard := r.shards.GetShard(r.registryId)
	err = shard.AddNode(uid, info)
	if err != nil {

	}

	return err
}

func (r *RegistryStorageRedis) RemoveNode(uid string) error {
	var err error

	shard := r.shards.GetShard(r.registryId)
	err = shard.RemoveNode(uid)
	if err != nil {

	}

	return err
}
