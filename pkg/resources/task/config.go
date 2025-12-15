package task

import "github.com/docker/go-connections/nat"

type Config struct {
	Name          string
	AttachStdin   bool
	AttachStdout  bool
	AttachStderr  bool
	ExposedPorts  nat.PortSet
	Cmd           []string
	Image         string
	CPU           float64
	Memory        int64
	Disk          int64
	Env           []string
	RestartPolicy string
}

func NewConfig(t Task) Config {
	return Config {
		Name: t.Name,
		ExposedPorts: t.ExposedPorts,
		Image: t.Image,
		CPU: t.CPU,
		Memory: int64(t.Memory),
		Disk: int64(t.Disk),
		RestartPolicy: t.RestartPolicy,
	}	
}
