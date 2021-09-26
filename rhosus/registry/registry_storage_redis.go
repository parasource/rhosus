package registry

import (
	"errors"
	"github.com/gomodule/redigo/redis"
	"github.com/parasource/rhosus/rhosus/util"
	"net"
	"strconv"
	"time"
)

const (
	defaultReadTimeout    = time.Second
	defaultWriteTimeout   = time.Second
	defaultConnectTimeout = time.Second
	defaultPoolSize       = 256
)

var (
	registerNodeSource = `redis.call("hset", KEYS[1], ARGV[1], ARGV[2])`

	removeNodeSource = `redis.call("hdel", KEYS[1], ARGV[1])`

	getNodesSource = `return redis.call("hgetall", KEYS[1])`
)

type RegistryStorageRedisShardConfig struct {
	Host             string
	Port             int
	Password         string
	DB               int
	UseTLS           bool
	TLSSkipVerify    bool
	MasterName       string
	IdleTimeout      time.Duration
	PubSubNumWorkers int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	ConnectTimeout   time.Duration
}

type redisShard struct {
	registry *Registry
	storage  *RegistryStorageRedis
	config   RegistryStorageRedisShardConfig
	pool     *redis.Pool

	eventMessages   chan redis.Message
	commandMessages chan redis.Message

	registerNodeScript *redis.Script
	removeNodeScript   *redis.Script
	getNodesScript     *redis.Script

	isAvailable       bool
	lastSeenAvailable time.Time
}

func NewShard(r *Registry, conf RegistryStorageRedisShardConfig) (*redisShard, error) {
	shard := &redisShard{
		registry: r,
		config:   conf,
		pool: func(registry *Registry, conf RegistryStorageRedisShardConfig) *redis.Pool {
			host := conf.Host
			port := conf.Port
			password := conf.Password
			db := conf.DB

			serverAddr := net.JoinHostPort(host, strconv.Itoa(port))

			poolSize := defaultPoolSize

			maxIdle := 64

			return &redis.Pool{
				MaxIdle:     maxIdle,
				MaxActive:   poolSize,
				Wait:        true,
				IdleTimeout: conf.IdleTimeout,
				Dial: func() (redis.Conn, error) {
					var err error

					var readTimeout = defaultReadTimeout
					if conf.ReadTimeout != 0 {
						readTimeout = conf.ReadTimeout
					}
					var writeTimeout = defaultWriteTimeout
					if conf.WriteTimeout != 0 {
						writeTimeout = conf.WriteTimeout
					}
					var connectTimeout = defaultConnectTimeout
					if conf.ConnectTimeout != 0 {
						connectTimeout = conf.ConnectTimeout
					}

					opts := []redis.DialOption{
						redis.DialConnectTimeout(connectTimeout),
						redis.DialReadTimeout(readTimeout),
						redis.DialWriteTimeout(writeTimeout),
					}
					c, err := redis.Dial("tcp", serverAddr, opts...)
					if err != nil {
						return nil, err
					}

					if password != "" {
						if _, err := c.Do("AUTH", password); err != nil {
							c.Close()
							return nil, err
						}
					}

					if db != 0 {
						if _, err := c.Do("SELECT", db); err != nil {
							c.Close()
							return nil, err
						}
					}

					return c, err
				},
				TestOnBorrow: func(c redis.Conn, t time.Time) error {
					_, err := c.Do("PING")
					return err
				},
			}
		}(r, conf),

		eventMessages:   make(chan redis.Message),
		commandMessages: make(chan redis.Message),

		registerNodeScript: redis.NewScript(1, registerNodeSource),
		removeNodeScript:   redis.NewScript(1, removeNodeSource),
		getNodesScript:     redis.NewScript(1, getNodesSource),
	}
	return shard, nil
}

// RegistryStorageRedis ///////////////////////
// Registry Storage that stores mappings files and bytes in Redis database
type RegistryStorageRedis struct {
	Registry *Registry

	shards []*redisShard
}

func NewRedisRegistryStorage(registry *Registry, shardsConf []RegistryStorageRedisShardConfig) (*RegistryStorageRedis, error) {

	var shards []*redisShard

	if shardsConf == nil || len(shardsConf) == 0 {
		return nil, errors.New("redis shards are not specified. either use different driver, or specify redis shards")
	}

	for _, conf := range shardsConf {
		shard, err := NewShard(registry, conf)
		if err != nil {
			return nil, err
		}
		shards = append(shards, shard)
	}

	return &RegistryStorageRedis{
		Registry: registry,
		shards:   shards,
	}, nil
}

func (r *RegistryStorageRedis) Run() error {

	for _, shard := range r.shards {
		go util.RunForever(func() {
			shard.run()
		})
	}

	return nil
}

func (s *redisShard) run() {
	var err error
	conn := s.pool.Get()

	err = s.registerNodeScript.Load(conn)
	if err != nil {

	}
	err = s.removeNodeScript.Load(conn)
	if err != nil {

	}
	err = s.getNodesScript.Load(conn)
	if err != nil {

	}

	err = conn.Close()
	if err != nil {

	}

	// todo pipeline
}

func (s *RegistryStorageRedis) RegisterNode(key string, data map[string]interface{}) error {

	return nil
}

func (s *RegistryStorageRedis) RemoveNode(key string) error {

	return nil
}

func (s *RegistryStorageRedis) GetNodes() map[string]map[string]interface{} {
	return nil
}
