package rhosus_redis

import (
	"errors"
	"github.com/gomodule/redigo/redis"
	rlog "github.com/parasource/rhosus/rhosus/logging"
	"github.com/parasource/rhosus/rhosus/registry"
	"github.com/parasource/rhosus/rhosus/util"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	defaultReadTimeout    = time.Second
	defaultWriteTimeout   = time.Second
	defaultConnectTimeout = time.Second
	defaultPoolSize       = 256
)

type RedisShardPool struct {
	registry *registry.Registry
	shards   []*RedisShard

	handler func(message redis.Message)
}

func NewRedisShardPool(registry *registry.Registry, conf []RedisShardConfig) (*RedisShardPool, error) {
	var shards []*RedisShard

	if conf == nil || len(conf) == 0 {
		return nil, errors.New("redis shards are not specified. either use different driver, or specify redis shards")
	}

	for _, conf := range conf {
		shard, err := NewShard(registry, conf)
		if err != nil {
			return nil, err
		}
		shards = append(shards, shard)
	}

	return &RedisShardPool{
		registry: registry,
		shards:   shards,
	}, nil
}

func (p *RedisShardPool) Run() {
	for _, shard := range p.shards {
		go util.RunForever(func() {
			shard.Run()
		})
	}
}

type RedisShardConfig struct {
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

type RedisShard struct {
	mu sync.RWMutex

	registry *registry.Registry
	config   RedisShardConfig

	shardsPool *RedisShardPool
	pool       *redis.Pool

	pubCh chan PubRequest
	recCh chan redis.Message

	isAvailable       bool
	lastSeenAvailable time.Time
}

func NewShard(r *registry.Registry, conf RedisShardConfig) (*RedisShard, error) {
	shard := &RedisShard{
		registry: r,
		config:   conf,
		pool: func(registry *registry.Registry, conf RedisShardConfig) *redis.Pool {
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

		pubCh: make(chan PubRequest),
		recCh: make(chan redis.Message),
	}
	return shard, nil
}

func (s *RedisShardPool) LoadScripts(scripts map[string]*redis.Script) error {

	var err error

	for _, shard := range s.shards {
		conn := shard.pool.Get()

		for name, script := range scripts {
			err := script.Load(conn)
			if err != nil {
				s.registry.Log.Log(rlog.NewLogEntry(rlog.LogLevelError, "error loading script "+name, map[string]interface{}{"error": err.Error()}))
			}
		}

		err = conn.Close()
		if err != nil {

		}
	}

	return err
}

func (s *RedisShard) Run() {

	go s.RunReceivePipeline()
	go s.RunPubPipeline()

}

func (s *RedisShardPool) SetMessagesHandler(handler func(message redis.Message)) {
	s.handler = handler
}
