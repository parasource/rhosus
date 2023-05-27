package profiler

import (
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"time"
)

type Profiler struct{}

func NewProfiler() (*Profiler, error) {
	return &Profiler{}, nil
}

func (p *Profiler) GetMem() (*mem.VirtualMemoryStat, error) {
	return mem.VirtualMemory()
}

func (p *Profiler) GetCPU() []float64 {
	i, err := cpu.Percent(time.Second, true)
	if err != nil {

	}

	return i
}

func (p *Profiler) GetDiskPartitionsUsage() (map[string]*disk.UsageStat, error) {
	disks, err := disk.Partitions(true)
	if err != nil {
		return nil, err
	}

	disksUsage := make(map[string]*disk.UsageStat)

	for i := range disks {
		usage, err := disk.Usage(disks[i].Mountpoint)
		if err != nil {
			continue
		}

		disksUsage[disks[i].Mountpoint] = usage
	}

	return disksUsage, nil
}

func (p *Profiler) GetPathDiskUsage(path string) *disk.UsageStat {
	stat, _ := disk.Usage(path)
	return stat
}
