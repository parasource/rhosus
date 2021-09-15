package stats

import (
	"syscall"
)

type DiskStats struct {
	All         uint64
	Free        uint64
	Used        uint64
	PercentFree float32
	PercentUsed float32
}

func GetDiskStats(dir string) (*DiskStats, error) {

	fs := syscall.Statfs_t{}
	err := syscall.Statfs(dir, &fs)
	if err != nil {
		return nil, err
	}
	stats := &DiskStats{
		All:  fs.Blocks * uint64(fs.Bsize),
		Free: fs.Bfree * uint64(fs.Bsize),
	}
	stats.Used = stats.All - stats.Free
	stats.PercentFree = float32((float64(stats.Free) / float64(stats.All)) * 100)
	stats.PercentUsed = float32((float64(stats.Used) / float64(stats.All)) * 100)

	return stats, nil
}
