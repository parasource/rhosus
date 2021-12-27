package rhosus_node

import (
	"github.com/parasource/rhosus/rhosus/util/tickers"
	"github.com/sirupsen/logrus"
	"time"
)

type StatsManager struct {
	node *Node
}

func NewStatsManager(node *Node) *StatsManager {
	return &StatsManager{
		node: node,
	}
}

func (s *StatsManager) Run() {

	ticker := tickers.SetTicker(time.Second * 5)

	for {
		select {
		case <-ticker.C:

			//_, err := s.node.grpc.NodeStats(context.Background(), &node_pb.NodeStatsRequest{
			//	Uid:  	"",
			//	Info: 	&node_pb.NodeInfo{
			//		Capacity:             0,
			//		Remaining:            0,
			//		LastUpdate:           0,
			//		Location:             "",
			//		State:                0,
			//		CacheCapacity:        0,
			//		CacheUsed:            0,
			//	},
			//})
			//if err != nil {
			//	continue
			//}

			logrus.Infof("node stats sended to registry")

		}
	}
}
