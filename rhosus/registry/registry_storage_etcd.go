package registry

import (
	rhosus_etcd "github.com/parasource/rhosus/rhosus/etcd"
	"github.com/sirupsen/logrus"
)

type EtcdStorageConfig struct {
}

type EtcdStorage struct {
	Config EtcdStorageConfig

	Registry   *Registry
	etcdClient *rhosus_etcd.EtcdClient
}

func NewEtcdStorage(conf EtcdStorageConfig, registry *Registry) (*EtcdStorage, error) {

	etcdClient, err := rhosus_etcd.NewEtcdClient(rhosus_etcd.EtcdClientConfig{
		Host: "localhost",
		Port: "2379",
	})
	if err != nil {
		logrus.Fatalf("error creating etcd client: %v", err)
	}

	return &EtcdStorage{
		Registry: registry,
		Config:   conf,

		etcdClient: etcdClient,
	}, nil
}

func (s *EtcdStorage) RegisterBlocks() {
	// just to show what this struct does
}
