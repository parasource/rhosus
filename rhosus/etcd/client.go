package rhosus_etcd

import (
	"context"
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	transmission_pb "github.com/parasource/rhosus/rhosus/pb/transmission"
	"github.com/parasource/rhosus/rhosus/util"
	"github.com/sirupsen/logrus"
	etcd "go.etcd.io/etcd/client/v3"
	"net"
	"strings"
	"time"
)

const (
	serviceDiscoveryRegistriesPath = "rhosus/service_discovery/registries/"
	serviceDiscoveryNodesPath      = "rhosus/service_discovery/nodes/"
	BlocksIndexesPath              = "rhosus/data/"
)

type EtcdClientConfig struct {
	Host string
	Port string
}

type EtcdClient struct {
	Config EtcdClientConfig

	cli *etcd.Client
}

func NewEtcdClient(conf EtcdClientConfig) (*EtcdClient, error) {

	client := &EtcdClient{
		Config: conf,
	}

	address := net.JoinHostPort(conf.Host, conf.Port)
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:   []string{address},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	client.cli = etcdClient

	return client, nil
}

func (c *EtcdClient) GetExistingNodes() (map[string][]byte, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	res, err := c.cli.Get(ctx, serviceDiscoveryNodesPath, etcd.WithPrefix())
	if err != nil {
		return nil, err
	}

	nodes := make(map[string][]byte)
	for _, kv := range res.Kvs {
		key := string(kv.Key)
		value := string(kv.Value)

		data, err := util.Base64Decode(value)
		if err != nil {

		}
		nodes[key] = data
	}

	return nodes, nil
}

func (c *EtcdClient) RegisterRegistry(uid string, info *registry_pb.RegistryInfo) error {
	path := serviceDiscoveryRegistriesPath + uid

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bytes, err := info.Marshal()
	if err != nil {
		logrus.Errorf("error marshaling registry info: %v", err)
	}

	_, err = c.cli.Put(ctx, path, util.Base64Encode(bytes))
	if err != nil {
		return err
	}

	return nil
}

func (c *EtcdClient) RegisterNode(name string, info *transmission_pb.NodeInfo) error {
	path := serviceDiscoveryNodesPath + name

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bytes, err := info.Marshal()
	if err != nil {
		logrus.Errorf("error marshaling registry info: %v", err)
	}

	_, err = c.cli.Put(ctx, path, util.Base64Encode(bytes))
	if err != nil {
		return err
	}

	return nil
}

func (c *EtcdClient) UnregisterRegistry(name string) error {
	path := serviceDiscoveryRegistriesPath + name

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := c.cli.Delete(ctx, path)
	return err
}

func (c *EtcdClient) UnregisterNode(name string) error {
	path := serviceDiscoveryNodesPath + name

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := c.cli.Delete(ctx, path)
	return err
}

func (c *EtcdClient) Put(ctx context.Context, key string, value string, ops ...etcd.OpOption) (*etcd.PutResponse, error) {
	return c.cli.Put(ctx, key, value, ops...)
}

func (c *EtcdClient) Get(ctx context.Context, key string, ops ...etcd.OpOption) (*etcd.GetResponse, error) {
	return c.cli.Get(ctx, key, ops...)
}

func (c *EtcdClient) WatchForRegistriesUpdates() etcd.WatchChan {
	return c.cli.Watch(context.Background(), serviceDiscoveryRegistriesPath, etcd.WithPrefix())
}

func (c *EtcdClient) WatchForNodesUpdates() etcd.WatchChan {
	return c.cli.Watch(context.Background(), serviceDiscoveryNodesPath, etcd.WithPrefix())
}

func ParseNodeName(path string) string {
	return strings.TrimPrefix(path, serviceDiscoveryNodesPath)
}

func ParseRegistryName(path string) string {
	return strings.TrimPrefix(path, serviceDiscoveryRegistriesPath)
}
