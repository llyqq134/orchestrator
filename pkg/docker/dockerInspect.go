package docker

import (
	"github.com/docker/docker/api/types/container"
)

type DockerInspectResponse struct {
	Error     error
	Container *container.InspectResponse
}
