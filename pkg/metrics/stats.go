package metrics

import (
	"log"

	"github.com/c9s/goprocinfo/linux"
	linuxproc "github.com/c9s/goprocinfo/linux"
)

type Stats struct {
	MemStats *linuxproc.MemInfo
	DiskStats *linuxproc.Disk
	CpuStats *linuxproc.CPUStat
	LoadStat *linuxproc.LoadAvg
	TaskCount int
}

func GetStats() *Stats {
	return &Stats {
		MemStats: GetMemoryInfo(),
		DiskStats: GetDiskInfo(),
		CpuStats: GetCpuInfo(),
		LoadStat: GetLoadAvg(),
	}
}

func GetMemoryInfo() *linuxproc.MemInfo {
	memstats, err := linuxproc.ReadMemInfo("/proc/meminfo")
	if err != nil {
		log.Printf("Error reading from /proc/meminfo")
		return &linuxproc.MemInfo{} 
	}

	return memstats 
}

func GetDiskInfo() *linuxproc.Disk {
	diskstats, err := linuxproc.ReadDisk("/")
	if err != nil {
		log.Printf("Error reading from /")
		return &linuxproc.Disk{}
	}

	return diskstats
}

func GetCpuInfo() *linuxproc.CPUStat {
	stats, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		log.Printf("Error reading from /proc/stat")
		return &linuxproc.CPUStat{}
	}

	return &stats.CPUStatAll
}

func GetLoadAvg() *linuxproc.LoadAvg {
	loadavg, err := linux.ReadLoadAvg("/proc/loadavg")
	if err != nil {
		log.Printf("Error reading from /proc/loadavg")
		return &linuxproc.LoadAvg{}
	}

	return loadavg
}

func (s *Stats) MemTotalKb() uint64 {
	return s.MemStats.MemTotal
}

func (s *Stats) MemAvailableKb() uint64 {
	return s.MemStats.MemAvailable
}

func (s *Stats) MemUsedKb() uint64 {
	return s.MemStats.MemTotal - s.MemStats.MemAvailable
}

func (s *Stats) MemUsedPercent() uint64 {
	return s.MemStats.MemAvailable / s.MemStats.MemTotal
}

func (s *Stats) DiskTotal() uint64 {
	return s.DiskStats.All 
}

func (s *Stats) DiskFree() uint64 {
	return s.DiskStats.Free 
}

func (s *Stats) DiskUsed() uint64 {
	return s.DiskStats.Used
}

func (s *Stats) CpuUsage() float64 {
	idle := s.CpuStats.Idle + s.CpuStats.IOWait
	nonIdle := s.CpuStats.User + s.CpuStats.Nice + s.CpuStats.System + s.CpuStats.IRQ + 
		s.CpuStats.SoftIRQ + s.CpuStats.Steal

		total := idle + nonIdle 

		if total == 0 {
			return 0.00
		}

		return (float64(total) - float64(idle)) / float64(total)
}
