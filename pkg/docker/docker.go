package docker

import (
	"context"
	"io"
	"log"
	"math"
	"orchestrator/pkg/resources/task"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

type Docker struct {
	Client *client.Client
	Config task.Config
}

type Result struct {
	Error       error
	Action      string
	ContainerID string
	Result      string
}

func NewDocker(c task.Config) *Docker {
	dc, _ := client.NewClientWithOpts(client.FromEnv)

	return &Docker {
		Client: dc,
		Config: c,
	}
}

func (d *Docker) Run() Result {
	op := "docker.Run: "

	ctx := context.Background()
	reader, err := d.Client.ImagePull(ctx, d.Config.Image, image.PullOptions{})
	if err != nil {
		log.Printf(op + "Failed to pull image: %v", err)
		return Result{Error: err}
	}
	io.Copy(os.Stdout, reader)

	rp := container.RestartPolicy {
		Name: container.RestartPolicyMode(d.Config.RestartPolicy),
	}

	r := container.Resources {
		Memory: d.Config.Memory,
		NanoCPUs: int64(d.Config.CPU * math.Pow(10, 9)),
	}

	cc := container.Config {
		Image: d.Config.Image,
		Tty: false,
		Env: d.Config.Env,
		ExposedPorts: d.Config.ExposedPorts,
	}

	hc := container.HostConfig {
		RestartPolicy: rp,
		Resources: r,
		PublishAllPorts: true,
	}

	resp, err := d.Client.ContainerCreate(ctx, &cc, &hc, nil, nil, d.Config.Name)
	if err != nil {
		log.Printf(op + "Error creating container using image %s: %v\n", d.Config.Image, err)
		return Result{Error: err}
	}

	if err = d.Client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		log.Printf(op + "Error starting container %s: %v\n", resp.ID, err)
		return Result{Error: err}
	}
	
	out, err := d.Client.ContainerLogs(
		ctx, 
		resp.ID,
		container.LogsOptions{ShowStdout: true, ShowStderr: true},
	)
	if err != nil {
		log.Printf(op + "Error getting logs for container %s: %v\n", resp.ID, err)
		return Result{Error: err}
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, out)

	return Result {ContainerID: resp.ID, Action: "start", Result: "success"}
}

func (d *Docker) Stop(id string) Result {
	op := "docker.Stop: "

	log.Printf(op + "Attempting to stop container %v", id )

	if err := d.Client.ContainerStop(context.Background(), id, container.StopOptions{});
		err != nil {
			log.Printf(op + "Error stopping container %s: %v\n", id, err)
			return Result{Error: err}
	}

	err := d.Client.ContainerRemove(context.Background(), id, container.RemoveOptions{
		RemoveVolumes: true,
		RemoveLinks: false,
		Force: false,
	})

	if err != nil {
		log.Printf(op + "Error removing container %s: %v\n", id, err)
		return Result{Error: err}
	}

	return Result{Action: "stop", Result: "success", Error: nil}
}
