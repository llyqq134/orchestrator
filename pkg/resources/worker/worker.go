package worker

import (
	"errors"
	"fmt"
	"log"
	"time"

	"orchestrator/pkg/docker"
	"orchestrator/pkg/metrics"
	"orchestrator/pkg/resources/task"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
	Stats     *metrics.Stats
}

func (w *Worker) CollectStats() {
	op := "[worker.CollectStats]: "
	for {
		log.Println(op + "Collecting stats\n")
		w.Stats = metrics.GetStats()
		w.Stats.TaskCount = w.TaskCount
		timeToSleep := 10
		time.Sleep(time.Duration(timeToSleep) * time.Second)
	}
}

func (w *Worker) GetTasks() []*task.Task {
	tasks := []*task.Task{}

	for _, t := range w.Db {
		tasks = append(tasks, t)
	}

	return tasks
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) runTask() docker.Result {
	op := "[worker.RunTask]: "
	t := w.Queue.Dequeue()
	if t == nil {
		log.Println(op + "No tasks in the queue")
		return docker.Result{Error: nil}
	}

	taskQueued := t.(task.Task)

	taskPersisted := w.Db[taskQueued.UUID]
	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db[taskQueued.UUID] = &taskQueued
	}

	var result docker.Result
	if task.ValidateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		case task.Completed:
			result = w.StopTask(taskQueued)
		default:
			result.Error = errors.New(op + "unreachable")
		}
	} else {
		err := fmt.Errorf(op+"Invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		result.Error = err
	}

	return result
}

func (w *Worker) StartTask(t task.Task) docker.Result {
	op := "[worker.StartTask]: "
	t.StartTime = time.Now().UTC()

	config := task.NewConfig(&t)
	d := docker.NewDocker(config)

	result := d.Run()
	if result.Error != nil {
		log.Printf(op+"Error running task %v: %v\n", t.UUID, result.Error)
		task.StateFailed(&t)
	} else {
		t.ContainerID = result.ContainerID
		task.StateRunning(&t)
	}

	w.Db[t.UUID] = &t

	return result
}

func (w *Worker) StopTask(t task.Task) docker.Result {
	op := "[worker.StopTask]: "
	config := task.NewConfig(&t)
	d := docker.NewDocker(config)

	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		log.Printf(op+"Error stopping container %v: %v\n", t.ContainerID, result.Error)
	}

	t.FinishTime = time.Now().UTC()
	task.StateCompleted(&t)
	w.Db[t.UUID] = &t

	log.Printf(op+"Stopped and removed container %v for task %v\n", t.ContainerID, t.UUID)

	return result
}

func (w *Worker) RunTasks() {
	op := "[worker.RunTasks]: "
	for {
		if w.Queue.Len() != 0 {
			result := w.runTask()
			if result.Error != nil {
				log.Printf(op+"Error running task: %v\n", result.Error)
			}
		} else {
			log.Println(op + "No tasks to process currently")
		}

		log.Println(op + "waiting for 10 sec")
		time.Sleep(time.Second * 10)
	}
}

func (w *Worker) InspectTask(t task.Task) docker.DockerInspectResponse {
	d := docker.NewDocker(task.NewConfig(&t))

	return d.Inspect(t.ContainerID)
}

func (w *Worker) updateTasks() {
	op := "[worker.updateTasks]: "
	for id, t := range w.Db {
		if t.State == task.Running {
			resp := w.InspectTask(*t)
			if resp.Error != nil {
				fmt.Printf("%vERROR: %v\n", op, resp.Error)
			}

			if resp.Container == nil {
				log.Printf(op+"No container for running task %s\n", id)
				task.StateFailed(w.Db[id])
			}

			if resp.Container.State.Status == "exited" {
				log.Printf(op+"Container for task %s in non running state %s",
					id, resp.Container.State.Status)
				task.StateFailed(w.Db[id])
			}

			w.Db[id].HostPorts =
				resp.Container.NetworkSettings.NetworkSettingsBase.Ports
		}
	}
}

func (w *Worker) UpdateTasks() {
	op := "[worker.UpdateTasks]: "
	for {
		log.Println(op + "Checking status of tasks")
		w.updateTasks()
		log.Println(op + "Task updates complete")

		timeToSleep := 10
		log.Printf(op+"Sleeping for %v seconds\n", timeToSleep)
		time.Sleep(time.Duration(timeToSleep) * time.Second)
	}
}
