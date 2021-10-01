package rhosus_redis

import (
	"github.com/parasource/rhosus/rhosus/util/timers"
	"time"
)

type PubRequest struct {
	Channel string
	Data    []byte
	Err     chan error
}

func (pr *PubRequest) done(err error) {
	pr.Err <- err
}

func (pr *PubRequest) result() error {
	return <-pr.Err
}

func (s *RedisShard) sendPubRequest(pub PubRequest) error {
	select {
	case s.pubCh <- pub:
	default:
		timer := timers.SetTimer(time.Second)
		defer timers.ReleaseTimer(timer)
		select {
		case s.pubCh <- pub:
		case <-timer.C:
			return RedisWriteTimeoutError
		}
	}

	return pub.result()
}
