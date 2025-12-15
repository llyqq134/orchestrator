package worker

import (
	"fmt"
	"log"
	"time"
	"errors"

	"orchestrator/pkg/docker"
	"orchestrator/pkg/resources/task"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        map[uuid.UUID]*task.Task
	TaskCount int
}

func (w *Worker) GetStats() {
	fmt.Println("collecting stats")
}

func (w *Worker) AddTask(t task.Task) {
	w.Queue.Enqueue(t)
}

func (w *Worker) RunTask() docker.Result {
	op := "worker.RunTask: "
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
		err := fmt.Errorf(op + "Invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		result.Error = err
	}

	return result
}

func (w *Worker) StartTask(t task.Task) docker.Result {
	op := "worker.StartTask: "
	t.StartTime = time.Now().UTC()

	config := task.NewConfig(&t)
	d := docker.NewDocker(config)

	result := d.Run()
	if result.Error != nil {
		log.Printf(op + "Error running task %v: %v\n", t.UUID, result.Error)
		task.StateFailed(t)
	} else {
		t.ContainerID = result.ContainerID
		task.StateRunning(t)
	}

	w.Db[t.UUID] = &t

	return result
}

func (w *Worker) StopTask(t task.Task) docker.Result {
	op := "worker.StopTask: "
	config := task.NewConfig(&t)
	d := docker.NewDocker(config)

	result := d.Stop(t.ContainerID)
	if result.Error != nil {
		log.Printf(op + "Error stopping container %v: %v\n", t.ContainerID, result.Error)
	}

	task.StateCompleted(t)
	w.Db[t.UUID] = &t

	log.Printf(op + "Stopped and removerd container %v for task %v\n", t.ContainerID, t.UUID)

	return result
}
