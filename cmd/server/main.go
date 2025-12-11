package main

import (
	"fmt"
	"orchestrator/pkg/docker"
	"orchestrator/pkg/resources/manager"
	"orchestrator/pkg/resources/node"
	"orchestrator/pkg/resources/task"
	"orchestrator/pkg/resources/worker"
	"time"
	"os"

	"github.com/docker/docker/client"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

func main() {
	t := task.Task{
		UUID:   uuid.New(),
		Name:   "Task-1",
		State:  task.Pending,
		Image:  "Image-1",
		Memory: 1024,
		Disk:   1,
	}

	taskEvent := task.Event{
		UUID:      uuid.New(),
		State:     task.Pending,
		Timestamp: time.Now(),
		Task:      t,
	}

	fmt.Printf("task: %v\n", t)
	fmt.Printf("task event: %v\n", taskEvent)

	w := worker.Worker{
		Name:  "worker-1",
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	fmt.Printf("Worker: %v\n", w)

	w.GetStats()
	w.RunTask()
	w.StartTask()
	w.StopTask()

	m := manager.Manager{
		Pending: *queue.New(),
		TaskDb:  make(map[string][]*task.Task),
		EventDb: make(map[string][]*task.Event),
		Workers: []string{w.Name},
	}

	fmt.Printf("Manager: %v\n", m)

	m.SelectWorker()
	m.UpdateTasks()
	m.SendWork()

	n := node.Node{
		Name:   "node-1",
		IpAddr: "192.168.1.1",
		Cores:  4,
		Memory: 1024,
		Disk:   25,
		Role:   "worker",
	}

	fmt.Printf("node: %v\n", n)

	fmt.Printf("create a test container\n")
	dockerTask, createResult := createContainer()
	if createResult.Error != nil {
		fmt.Printf("%v\n", createResult.Error)
		os.Exit(1)
	}

	time.Sleep(time.Second * 5)
	fmt.Printf("stopping container %s\n", createResult.ContainerID)
	_ = stopContainer(dockerTask, createResult.ContainerID)
}

func createContainer() (*docker.Docker, *docker.Result) {
	c := task.Config {
		Name: "test-container-1",
		Image: "postgres:13",
		Env: []string {
			"POSTGRES_USER=cube",
			"POSTGRES_PASSWORD=secret",
		},
		RestartPolicy: "on-failure",
	}

	dc, _ := client.NewClientWithOpts(client.FromEnv)
	d := docker.Docker {
		Client: dc,
		Config: c,
	}

	result := d.Run()
	if result.Error != nil {
		fmt.Printf("%v\n", result.Error)
		stopContainer(&d, result.ContainerID)
		return nil, nil
	}

	fmt.Printf("Container %s is running with config %v\n", result.ContainerID, c)                      

	return &d, &result
}

func stopContainer(d *docker.Docker, id string) *docker.Result {
	result := d.Stop(id)
	if result.Error != nil {
		fmt.Printf("%v\n", result.Error)
		return nil
	}

	fmt.Printf("Container %s has benn stopped and removed\n", result.ContainerID)

	return &result
}
