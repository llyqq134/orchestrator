package metrics

import (
	linuxproc "github.com/c9s/goprocinfo/linux"
)

type Stats struct {
	Stats *linuxproc.MemInfo
	DiskStats *linuxproc.DiskStat 
	CpuStats *linuxproc.CPUStat
	LoadStat *linuxproc.LoadAvg
}
