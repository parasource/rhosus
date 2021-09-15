package storage

type MinFreeSpaceType int

const (
	Percent MinFreeSpaceType = iota
	Bytes
)

type MinFreeSpace struct {
	Type    MinFreeSpaceType
	Bytes   uint64
	Percent float32
	Raw     string
}

func (s *MinFreeSpace) IsLow(bytes uint64, percent float32) bool {
	switch s.Type {
	case Percent:
		return percent < s.Percent
	case Bytes:
		return bytes < s.Bytes
	}

	return false
}
