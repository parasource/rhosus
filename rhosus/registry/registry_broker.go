package registry

import rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"

type RegistryBroker interface {
	PublishCommand(pubReq rhosus_redis.PubRequest) error
}
