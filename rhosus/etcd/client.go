package rhosus_etcd

import (
	"context"
	"fmt"
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/parasource/rhosus/rhosus/util"
	etcd "go.etcd.io/etcd/client/v3"
	"strings"
	"time"
)

const (
	serviceDiscoveryRegistriesPath = "rhosus/service_discovery/registries/"
	serviceDiscoveryNodesPath      = "rhosus/service_discovery/nodes/"
)

type EtcdClientConfig struct {
	Address string `json:"address"`
	Timeout int    `json:"timeout"`
}

type EtcdClient struct {
	Config EtcdClientConfig

	cli *etcd.Client
}

func NewEtcdClient(conf EtcdClientConfig) (*EtcdClient, error) {

	client := &EtcdClient{
		Config: conf,
	}

	address := strings.Split(conf.Address, ",")
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:   address,
		DialTimeout: time.Duration(conf.Timeout) * time.Second,
	})
	if err != nil {
		return nil, err
	}

	client.cli = etcdClient

	return client, nil
}

func (c *EtcdClient) Ping() error {

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := c.cli.Put(ctx, "ping", "pong")
	return err

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

func (c *EtcdClient) GetExistingRegistries() (map[string][]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	res, err := c.cli.Get(ctx, serviceDiscoveryRegistriesPath, etcd.WithPrefix())
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

func (c *EtcdClient) RegisterRegistry(uid string, info *control_pb.RegistryInfo) error {
	path := serviceDiscoveryRegistriesPath + uid

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bytes, err := info.Marshal()
	if err != nil {
		return fmt.Errorf("error marshaling registry info: %v", err)
	}

	_, err = c.cli.Put(ctx, path, util.Base64Encode(bytes))
	if err != nil {
		return err
	}

	return nil
}

func (c *EtcdClient) RegisterNode(id string, info *transport_pb.NodeInfo) error {
	path := serviceDiscoveryNodesPath + id

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bytes, err := info.Marshal()
	if err != nil {
		return fmt.Errorf("error marshaling datanode info: %v", err)
	}

	_, err = c.cli.Put(ctx, path, util.Base64Encode(bytes))
	if err != nil {
		return err
	}

	return nil
}

func (c *EtcdClient) UnregisterRegistry(uid string) error {
	path := serviceDiscoveryRegistriesPath + uid

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := c.cli.Delete(ctx, path)
	return err
}

func (c *EtcdClient) UnregisterNode(id string) error {
	path := serviceDiscoveryNodesPath + id

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

func (c *EtcdClient) Delete(ctx context.Context, key string, ops ...etcd.OpOption) (*etcd.DeleteResponse, error) {
	return c.cli.Delete(ctx, key, ops...)
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
