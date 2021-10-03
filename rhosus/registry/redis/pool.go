package rhosus_redis

import (
	"errors"
	"github.com/gomodule/redigo/redis"
	"github.com/parasource/rhosus/rhosus/util/timers"
	"github.com/sirupsen/logrus"
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
	Shards []*RedisShard

	handler func(message redis.Message)
}

func NewRedisShardPool(conf []RedisShardConfig) (*RedisShardPool, error) {
	var shards []*RedisShard

	pool := &RedisShardPool{}

	if conf == nil || len(conf) == 0 {
		return nil, errors.New("redis Shards are not specified. either use different driver, or specify redis Shards")
	}

	for _, conf := range conf {
		shard, err := NewShard(pool, conf)
		if err != nil {
			return nil, err
		}
		shards = append(shards, shard)
	}

	pool.Shards = shards

	return pool, nil
}

func (p *RedisShardPool) Run() {
	for _, shard := range p.Shards {
		shard.Run()
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

	config RedisShardConfig

	shardsPool *RedisShardPool
	pool       *redis.Pool

	pubCh chan PubRequest
	recCh chan redis.Message

	isAvailable       bool
	lastSeenAvailable time.Time
}

func NewShard(pool *RedisShardPool, conf RedisShardConfig) (*RedisShard, error) {
	shard := &RedisShard{
		shardsPool: pool,
		config:     conf,
		pool: func(conf RedisShardConfig) *redis.Pool {
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
		}(conf),

		pubCh: make(chan PubRequest),
		recCh: make(chan redis.Message),
	}
	return shard, nil
}

func (s *RedisShardPool) LoadScripts(scripts map[string]*redis.Script) error {

	var err error

	for _, shard := range s.Shards {
		conn := shard.pool.Get()

		for name, script := range scripts {
			err := script.Load(conn)
			if err != nil {
				logrus.Errorf("error loading script %v: %v", name, err)
			}
		}

		err = conn.Close()
		if err != nil {

		}
	}

	return err
}

func (s *RedisShard) Run() {

	go s.runPubPipeline()
	go s.runReceivePipeline()
	go s.runPingPipeline()

}

func (s *RedisShardPool) SetMessagesHandler(handler func(message redis.Message)) {
	s.handler = handler
}

func (s *RedisShard) Publish(pubReq PubRequest) error {
	select {
	case s.pubCh <- pubReq:
	default:
		timer := timers.SetTimer(time.Second * 5)
		select {
		case s.pubCh <- pubReq:
		case <-timer.C:
			return RedisWriteTimeoutError
		}
	}

	return pubReq.result()
}
