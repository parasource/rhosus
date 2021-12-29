package registry

import (
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
)

type RegistryBroker interface {
	PublishFileRegisteredEvent(ev *registry_pb.EventFileRegistered) error
}
