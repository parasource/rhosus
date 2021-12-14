package registry

import (
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	rhosus_redis "github.com/parasource/rhosus/rhosus/registry/redis"
	"github.com/sirupsen/logrus"
)

type RegistryBrokerRedis struct {
	config rhosus_redis.RedisConfig

	pool *rhosus_redis.RedisShardPool

	shutdownCh chan struct{}
}

func NewRegistryRedisBroker(shardsPool *rhosus_redis.RedisShardPool) (*RegistryBrokerRedis, error) {
	r := &RegistryBrokerRedis{
		pool:       shardsPool,
		shutdownCh: make(chan struct{}, 1),
	}

	return r, nil
}

func (r *RegistryBrokerRedis) Shutdown() error {
	var err error

	return err
}

func (b *RegistryBrokerRedis) PublishFileRegisteredEvent(ev *registry_pb.EventFileRegistered) error {
	bytes, err := ev.Marshal()
	if err != nil {
		logrus.Errorf("error marshaling event: %v", err)
	}

	event := &registry_pb.Event{
		Type: registry_pb.Event_FILE_REGISTERED,
		Data: bytes,
	}
	bytes, err = event.Marshal()
	if err != nil {
		logrus.Errorf("error marshaling event: %v", err)
	}

	req := rhosus_redis.PubRequest{
		Channel: rhosus_redis.EventsChannel,
		Data:    bytes,
		Err:     make(chan error, 1),
	}

	return b.Publish(req)
}

func (b *RegistryBrokerRedis) PublishFileDeletedEvent(ev *registry_pb.EventFileDeleted) error {
	bytes, err := ev.Marshal()
	if err != nil {
		logrus.Errorf("error marshaling event: %v", err)
	}

	event := &registry_pb.Event{
		Type: registry_pb.Event_FILE_DELETED,
		Data: bytes,
	}
	bytes, err = event.Marshal()
	if err != nil {
		logrus.Errorf("error marshaling event: %v", err)
	}

	req := rhosus_redis.PubRequest{
		Channel: rhosus_redis.EventsChannel,
		Data:    bytes,
		Err:     make(chan error, 1),
	}

	return b.Publish(req)
}

func (b *RegistryBrokerRedis) PublishFileModifiedEvent(ev *registry_pb.EventFileModified) error {
	bytes, err := ev.Marshal()
	if err != nil {
		logrus.Errorf("error marshaling event: %v", err)
	}

	event := &registry_pb.Event{
		Type: registry_pb.Event_FILE_MODIFIED,
		Data: bytes,
	}
	bytes, err = event.Marshal()
	if err != nil {
		logrus.Errorf("error marshaling event: %v", err)
	}

	req := rhosus_redis.PubRequest{
		Channel: rhosus_redis.EventsChannel,
		Data:    bytes,
		Err:     make(chan error, 1),
	}

	return b.Publish(req)
}

func (b *RegistryBrokerRedis) PublishRegistryShutdownEvent(ev *registry_pb.EventRegistryShutdown) error {
	bytes, err := ev.Marshal()
	if err != nil {
		logrus.Errorf("error marshaling event: %v", err)
	}

	event := &registry_pb.Event{
		Type: registry_pb.Event_REGISTRY_SHUTDOWN,
		Data: bytes,
	}
	bytes, err = event.Marshal()
	if err != nil {
		logrus.Errorf("error marshaling event: %v", err)
	}

	req := rhosus_redis.PubRequest{
		Channel: rhosus_redis.EventsChannel,
		Data:    bytes,
		Err:     make(chan error, 1),
	}

	return b.Publish(req)
}

func (r *RegistryBrokerRedis) Publish(pubReq rhosus_redis.PubRequest) error {
	return r.pool.Shards[0].Publish(pubReq)
}
