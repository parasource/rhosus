package data

type Manager struct {
	partitions *PartitionsMap
}

func NewManager() (*Manager, error) {
	pmap, err := NewPartitionsMap(defaultPartitionsDir, 1024)
	if err != nil {
		return nil, err
	}

	return &Manager{
		partitions: pmap,
	}, nil
}
