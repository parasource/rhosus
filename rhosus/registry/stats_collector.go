package registry

import (
	transport_pb "github.com/parasource/rhosus/rhosus/pb/transport"
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"sync"
	"time"
)

type StatsCollector struct {
	registry *Registry

	mu               sync.RWMutex
	metrics          map[string]*transport_pb.NodeMetrics
	metricsUpdatedAt map[string]time.Time

	collectionInterval int
}

func NewStatsCollector(registry *Registry, collectionInterval int) *StatsCollector {
	return &StatsCollector{
		registry:           registry,
		metrics:            make(map[string]*transport_pb.NodeMetrics),
		metricsUpdatedAt:   make(map[string]time.Time),
		collectionInterval: collectionInterval,
	}
}

func (s *StatsCollector) Run() {

	ticker := tickers.SetTicker(time.Second * time.Duration(s.collectionInterval))

	for {
		select {
		case <-ticker.C:

		}
	}
}
