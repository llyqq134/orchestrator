package node

type Node struct {
	Name            string
	IpAddr          string
	Cores           int
	Memory          int
	MemoryAllocated int
	Disk            int
	DiskAllocated   int
	Role            string
	TaskCount       int
}
