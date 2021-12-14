package rhosus_redis

import (
	"github.com/gomodule/redigo/redis"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"strings"
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
			} else {
				logrus.Fatalf("broker message handler is not set")
			}
		case redis.Subscription:
			//logrus.Infof("subscription received to %v", m.Channel)
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

func (s *RedisShard) runDataPipeline() {

	var err error

	conn := s.pool.Get()

	err = s.registerFileScript.Load(conn)
	if err != nil {
		logrus.Errorf("error adding redis register file script; %v", err)
		conn.Close()
		return
	}

	err = s.removeFileScript.Load(conn)
	if err != nil {
		logrus.Errorf("error adding redis remove file script; %v", err)
		conn.Close()
		return
	}

	err = s.addNodeScript.Load(conn)
	if err != nil {
		logrus.Errorf("error adding redis add node script; %v", err)
		conn.Close()
		return
	}

	err = s.removeNodeScript.Load(conn)
	if err != nil {
		logrus.Errorf("error adding redis remove node script; %v", err)
		conn.Close()
		return
	}

	err = conn.Close()
	if err != nil {

	}

	var drs []DataRequest

	for dr := range s.dataCh {
		drs = append(drs, dr)
	loop:
		for len(drs) < 512 {
			select {
			case dr := <-s.dataCh:
				drs = append(drs, dr)
			default:
				break loop
			}
		}

		conn := s.pool.Get()

		for i := range drs {
			switch drs[i].op {
			case DataOpRegisterFile:
				err := s.registerFileScript.SendHash(conn, drs[i].args...)
				if err != nil {

				}
			case DataOpRemoveFile:
				err := s.removeFileScript.SendHash(conn, drs[i].args...)
				if err != nil {

				}
			case DataOpAddNode:
				err := s.addNodeScript.SendHash(conn, drs[i].args...)
				if err != nil {

				}
			case DataOpRemoveNode:
				err := s.removeNodeScript.SendHash(conn, drs[i].args...)
				if err != nil {

				}
			}
		}

		err := conn.Flush()

		if err != nil {
			for i := range drs {
				drs[i].done(nil, err)
			}
			logrus.Errorf("error flushing data pipeline: %v", err)
		}

		var noScriptError bool
		for i := range drs {
			reply, err := conn.Receive()
			if err != nil {
				if e, ok := err.(redis.Error); ok && strings.HasPrefix(string(e), "NOSCRIPT ") {
					noScriptError = true
				}
			}
			drs[i].done(reply, err)
		}
		if noScriptError {
			// Start this func from the beginning and LOAD missing script.
			conn.Close()
			return
		}
		conn.Close()
		drs = nil
	}

}
