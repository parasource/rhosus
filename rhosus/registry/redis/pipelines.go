package rhosus_redis

import (
	"fmt"
	"time"
)

func (s *RedisShard) RunReceivePipeline() {
	for {
		select {
		case <-s.registry.NotifyShutdown():
			return
		case m := <-s.recCh:
			if s.shardsPool.handler != nil {
				s.shardsPool.handler(m)
			}

			fmt.Printf("received redis from Channel %v with Data %v", m.Channel, m.Data)
		}
	}
}

func (s *RedisShard) RunPubPipeline() {
	var prs []PubRequest

	pingTicker := time.NewTicker(time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-s.registry.NotifyShutdown():
			return
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
