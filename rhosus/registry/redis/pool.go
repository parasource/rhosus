package rhosus_redis

import (
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	node_pb "github.com/parasource/rhosus/rhosus/pb/node"
	registry_pb "github.com/parasource/rhosus/rhosus/pb/registry"
	"github.com/parasource/rhosus/rhosus/util"
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
	Shards  []*RedisShard
	readyCh chan struct{}

	handler func(message redis.Message)
}

func NewRedisShardPool(config RedisConfig) (*RedisShardPool, error) {

	var shardsConfig []RedisShardConfig

	numShards := 0
	if len(config.Hosts) > numShards {
		numShards = len(config.Hosts)
	}
	if len(config.Ports) > numShards {
		numShards = len(config.Ports)
	}
	for i := 0; i < numShards; i++ {
		port, err := strconv.Atoi(config.Ports[i])
		if err != nil {
			return nil, fmt.Errorf("malformed port: %v", err)
		}
		conf := RedisShardConfig{
			Host:          config.Hosts[i],
			Port:          port,
			Password:      config.Password,
			DB:            config.DB,
			UseTLS:        config.UseTLS,
			TLSSkipVerify: config.TLSSkipVerify,
			MasterName:    config.MasterName,
			IdleTimeout:   config.IdleTimeout * time.Second,
			//ConnectTimeout:   time.Duration(v.GetInt("redis_connect_timeout")) * time.Second,
			//ReadTimeout:      time.Duration(v.GetInt("redis_read_timeout")) * time.Second,
			//WriteTimeout:     time.Duration(v.GetInt("redis_write_timeout")) * time.Second,
		}
		shardsConfig = append(shardsConfig, conf)
	}

	var shards []*RedisShard

	pool := &RedisShardPool{
		readyCh: make(chan struct{}),
	}

	if shardsConfig == nil || len(shardsConfig) == 0 {
		return nil, errors.New("redis shards are not specified. either use different driver, or specify redis Shards")
	}

	for _, conf := range shardsConfig {
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
	allOk := true

	shardsOk := 0
	for _, shard := range p.Shards {
		if err := shard.Run(); err != nil {
			logrus.Errorf("error connecting to redis shard: %v", err)

			allOk = false
		} else {
			shardsOk++
		}
	}

	if shardsOk < 1 {
		logrus.Fatal("No alive redis shards. Check your config")
	}

	if !allOk {
		logrus.Warnf("error occured while connecting to one of shards")
	} else {
		logrus.Infof("Successfully connected to redis shards")
	}

	close(p.readyCh)
}

func (p *RedisShardPool) NotifyReady() <-chan struct{} {
	return p.readyCh
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
	id int
	mu sync.RWMutex

	config RedisShardConfig

	shardsPool *RedisShardPool
	pool       *redis.Pool

	pubCh  chan PubRequest
	recCh  chan redis.Message
	dataCh chan DataRequest

	registerFileScript *redis.Script
	removeFileScript   *redis.Script
	addNodeScript      *redis.Script
	removeNodeScript   *redis.Script
	getNodesScript     *redis.Script

	isAvailable       bool
	lastSeenAvailable time.Time
}

func NewShard(pool *RedisShardPool, conf RedisShardConfig) (*RedisShard, error) {
	shard := &RedisShard{
		shardsPool: pool,
		config:     conf,

		registerFileScript: redis.NewScript(1, registerFileSource),
		removeFileScript:   redis.NewScript(1, removeFileSource),
		addNodeScript:      redis.NewScript(1, addNodeSource),
		removeNodeScript:   redis.NewScript(1, removeNodeSource),
		getNodesScript:     redis.NewScript(1, getNodesSource),

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

func (s *RedisShard) Run() error {

	// Test the connection
	err := s.testConnection()
	if err != nil {
		return err
	}

	go s.runPubPipeline()
	go s.runReceivePipeline()
	go s.runDataPipeline()
	go s.runPingPipeline()

	port := strconv.Itoa(s.config.Port)
	logrus.Infof("Connected to redis shard on %v", net.JoinHostPort(s.config.Host, port))

	return nil
}

func (s *RedisShard) testConnection() error {
	conn := s.pool.Get()
	defer conn.Close()

	_, err := conn.Do("PING")
	return err
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

func (s *RedisShard) RegisterFile(id string, info *registry_pb.FileInfo) error {

	bytes, err := info.Marshal()
	if err != nil {
		logrus.Errorf("error marshaling file info: %v", err)
	}
	dr := newDataRequest(DataOpRegisterFile, []interface{}{id, bytes})
	res := s.getDataResponse(dr)

	return res.err
}

func (s *RedisShard) RemoveFile(id string) error {
	dr := newDataRequest(DataOpRemoveFile, []interface{}{id})
	res := s.getDataResponse(dr)

	return res.err
}

func (s *RedisShard) AddNode(uid string, info *node_pb.NodeInfo) error {
	bytes, err := info.Marshal()
	if err != nil {

	}

	dr := newDataRequest(DataOpAddNode, []interface{}{uid, bytes})
	res := s.getDataResponse(dr)

	return res.err

}

func (s *RedisShard) RemoveNode(uid string) error {
	dr := newDataRequest(DataOpRemoveNode, []interface{}{})
	res := s.getDataResponse(dr)

	return res.err
}

func (s *RedisShard) getDataResponse(r DataRequest) *DataResponse {
	select {
	case s.dataCh <- r:
	default:
		timer := timers.SetTimer(time.Second * 5)
		defer timers.ReleaseTimer(timer)
		select {
		case s.dataCh <- r:
		case <-timer.C:
			return &DataResponse{r.result(), errors.New("redis timeout")}
		}
	}
	return r.result()
}

type dataOp int

const (
	DataOpRegisterFile dataOp = iota
	DataOpRemoveFile
	DataOpAddNode
	DataOpRemoveNode
	DataOpUpdateNodeStats
)

type DataResponse struct {
	reply interface{}
	err   error
}

type DataRequest struct {
	op   dataOp
	args []interface{}
	resp chan *DataResponse
}

func newDataRequest(op dataOp, args []interface{}) DataRequest {
	return DataRequest{op: op, args: args, resp: make(chan *DataResponse, 1)}
}

func (dr *DataRequest) done(reply interface{}, err error) {
	if dr.resp == nil {
		return
	}
	dr.resp <- &DataResponse{reply: reply, err: err}
}

func (dr *DataRequest) result() *DataResponse {
	if dr.resp == nil {
		// No waiting, as caller didn't care about response.
		return &DataResponse{}
	}
	return <-dr.resp
}

func (p *RedisShardPool) GetShard(registryId string) *RedisShard {
	return p.Shards[util.ConsistentIndex(registryId, len(p.Shards))]
}
