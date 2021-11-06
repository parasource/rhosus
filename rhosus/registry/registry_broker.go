package registry

import (
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
)

type RegistryBroker interface {
	PublishFileRegisteredEvent(ev *registry_pb.EventFileRegistered) error
	Publish(pubReq rhosus_redis.PubRequest) error
}
