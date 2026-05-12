package node

import "orchestrator/pkg/metrics"

type Node struct {
	Name            string
	IpAddr          string
	Api             string
	Cores           int
	Memory          int
	MemoryAllocated int
	Disk            int
	DiskAllocated   int
	Stats           metrics.Stats
	Role            string
	TaskCount       int
}

func New(name, api, role string) *Node {
	return &Node{
		Name: name,
		Api:  api,
		Role: role,
	}
}
