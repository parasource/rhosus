package rhosus_redis

import (
	"github.com/gomodule/redigo/redis"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"time"
)

func (s *RedisShard) runReceivePipeline() {

	conn := s.pool.Get()

	err := conn.Err()
	if err != nil {
		logrus.Errorf("error dialing redis: %v", err)
	}

	psc := &redis.PubSubConn{Conn: conn}
	err = psc.Subscribe(RegistryInfoChannel, PingChannel)
	if err != nil {
		logrus.Errorf("error subscribing to channels: %v", err)
	}

	for {
		switch m := psc.ReceiveWithTimeout(time.Second * 10).(type) {
		case redis.Message:
			if s.shardsPool.handler != nil {
				s.shardsPool.handler(m)
			}
		case redis.Subscription:
			logrus.Infof("subscription received to %v", m.Channel)
		}
	}
}

func (s *RedisShard) runPubPipeline() {
	var prs []PubRequest

	for {
		select {
		//case <-s.registry.NotifyShutdown():
		//	return
		case p := <-s.pubCh:
			prs = append(prs, p)

		loop:
			for len(prs) < 512 {
				select {
				case pr := <-s.pubCh:
					prs = append(prs, pr)
				default:
					break loop
				}
			}
			conn := s.pool.Get()
			for i := range prs {
				err := conn.Send("PUBLISH", prs[i].Channel, prs[i].Data)
				if err != nil {

				}
				prs[i].done(nil)
			}
			err := conn.Flush()
			if err != nil {
				for i := range prs {
					prs[i].done(err)
					err := conn.Close()
					if err != nil {

					}
				}
			}
			err = conn.Close()
			if err != nil {

			}

			prs = nil
		}
	}
}

func (s *RedisShard) runPingPipeline() {
	ticker := tickers.SetTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			conn := s.pool.Get()

			err := conn.Send("PUBLISH", PingChannel, "ping")
			if err != nil {
				logrus.Errorf("error publishing to ping channel: %v", err)
			}

			conn.Close()
		}
	}
}
