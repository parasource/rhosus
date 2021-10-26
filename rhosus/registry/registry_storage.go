package registry

import (
	node_pb "github.com/parasource/rhosus/rhosus/pb/node"
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
)

type RegistryStorage interface {
	GetNodes()
	RegisterFile(id string, info *registry_pb.FileInfo) error
	RemoveFile(id string) error
	AddNode(uid string, info *node_pb.NodeInfo) error
	RemoveNode(uid string) error
}
