package registry

import (
	"context"
	transmission_pb "github.com/parasource/rhosus/rhosus/pb/transmission"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type StatsCollector struct {
	registry *Registry

	mu               sync.RWMutex
	metrics          map[string]*transmission_pb.NodeMetrics
	metricsUpdatedAt map[string]time.Time

	collectionInterval int
}

func NewStatsCollector(registry *Registry, collectionInterval int) *StatsCollector {
	return &StatsCollector{
		registry:           registry,
		metrics:            make(map[string]*transmission_pb.NodeMetrics),
		metricsUpdatedAt:   make(map[string]time.Time),
		collectionInterval: collectionInterval,
	}
}

func (s *StatsCollector) Run() {

	ticker := tickers.SetTicker(time.Second * time.Duration(s.collectionInterval))

	for {
		select {
		case <-ticker.C:

			nodes := make(map[string]*NodeInfo)

			s.mu.RLock()
			for uid, info := range s.registry.NodesMap.nodes {
				nodes[uid] = info
			}
			s.mu.RUnlock()

			for uid := range nodes {
				client, err := s.registry.NodesMap.GetGrpcClient(uid)
				if err != nil {
					logrus.Errorf("error getting node grpc conn: %v", err)
					continue
				}

				res, err := client.FetchMetrics(context.Background(), &transmission_pb.FetchMetricsRequest{})
				if err != nil {
					logrus.Errorf("error fetching node metrics: %v", err)
				}

				s.mu.Lock()
				if res.Uid == uid {
					s.metrics[uid] = res.Metrics
					s.metricsUpdatedAt[uid] = time.Unix(res.Metrics.LastUpdate, 0)
				}
				s.mu.Unlock()

				logrus.Infof("metrics has been updated for node %v", res.Uid)
			}
		}
	}
}
