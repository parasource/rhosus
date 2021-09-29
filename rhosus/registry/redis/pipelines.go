package rhosus_redis

import "fmt"

func (s *RedisShard) RunReceivePipeline() {
	for {
		select {
		case <-s.registry.NotifyShutdown():
			return
		case m := <-s.recCh:
			ch := m.Channel
			data := m.Data

			fmt.Printf("received redis from channel %v with data %v", ch, data)
		}
	}
}

func (s *RedisShard) RunPubPipeline() {
	for {
		select {
		case <-s.registry.NotifyShutdown():
			return
		case m := <-s.pubCh:
			fmt.Printf("message ready to rock: %v", m)
		}
	}
}
