package rhosus_node

type StatsManager struct {
	node *Node
}

func NewStatsManager(node *Node) *StatsManager {
	return &StatsManager{
		node: node,
	}
}
