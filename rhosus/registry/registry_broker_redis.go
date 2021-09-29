package registry

import (
	rlog "github.com/parasource/rhosus/rhosus/logging"
	"github.com/parasource/rhosus/rhosus/registry/redis"
)

var (
	registerNodeSource = `redis.call("hset", KEYS[1], ARGV[1], ARGV[2])`

	removeNodeSource = `redis.call("hdel", KEYS[1], ARGV[1])`

	getNodesSource = `return redis.call("hgetall", KEYS[1])`
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

	err := r.Run()
	if err != nil {
		r.registry.Log.Log(rlog.NewLogEntry(rlog.LogLevelError, "error running redis broker registry", map[string]interface{}{"error": err.Error()}))
	}

	return r, nil
}

func (r *RegistryBrokerRedis) Run() error {
	return nil
}
