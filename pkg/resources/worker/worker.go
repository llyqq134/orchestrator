package worker

import (
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"orchestrator/pkg/docker"
	"orchestrator/pkg/metrics"
	"orchestrator/pkg/resources/task"
	"orchestrator/pkg/store"

	"github.com/golang-collections/collections/queue"
)

type Worker struct {
	Name      string
	Queue     queue.Queue
	Db        store.Store
	TaskCount int
	Stats     *metrics.Stats
}

func New(name, taskDbType, dataDir string) *Worker {
	op := "[worker.New]: "

	w := Worker{
		Name:  name,
		Queue: *queue.New(),
	}

	var s store.Store
	var err error

	switch taskDbType {
	case store.PersistentStore:
		filename := filepath.Join(dataDir, fmt.Sprintf("%s_tasks.db", name))
		s, err = store.NewTaskStore(filename, 0600, "tasks")
	default:
		s = store.NewInMemoryTaskStore()
	}

	if err != nil {
		log.Fatalf(op+"unable to create task store: %v", err)
	}

	w.Db = s

	return &w
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
	op := "[worker.GetTasks]: "

	taskList, err := w.Db.List()
	if err != nil {
		log.Printf(op+"Error getting list of tasks: %v\n", err)
		return nil
	}

	return taskList.([]*task.Task)
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
	log.Printf(op+"Worker found task %v in the queue\n", taskQueued)

	err := w.Db.Put(taskQueued.UUID.String(), &taskQueued)
	if err != nil {
		log.Printf(op+"Error storing task %s: %v\n", taskQueued.UUID.String(), err)
		return docker.Result{Error: err}
	}

	result, err := w.Db.Get(taskQueued.UUID.String())
	if err != nil {
		log.Printf(op+"Error getting task %s from database: %v\n", taskQueued.UUID.String(), err)
		return docker.Result{Error: err}
	}

	taskPersisted := *result.(*task.Task)

	if taskPersisted.State == task.Completed {
		return w.StopTask(taskPersisted)
	}

	var dockerResult docker.Result
	if task.ValidateTransition(taskPersisted.State, taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			if taskQueued.ContainerID != "" {
				dockerResult = w.StopTask(taskQueued)
				if dockerResult.Error != nil {
					log.Printf(op+"%v\n", dockerResult.Error)
				}
			}
			dockerResult = w.StartTask(taskQueued)
		default:
			dockerResult.Error = errors.New(op + "unreachable")
		}
	} else {
		err := fmt.Errorf(op+"Invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
		dockerResult.Error = err
	}

	return dockerResult
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
		w.Db.Put(t.UUID.String(), &t)

		return result
	}

	t.ContainerID = result.ContainerID
	task.StateRunning(&t)
	w.Db.Put(t.UUID.String(), &t)

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
	w.Db.Put(t.UUID.String(), &t)

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

	tasks, err := w.Db.List()
	if err != nil {
		log.Printf(op+"Error getting list of tasks: %v\n", err)
		return
	}

	for id, t := range tasks.([]*task.Task) {
		if t.State == task.Running {
			resp := w.InspectTask(*t)
			if resp.Error != nil {
				log.Printf(op+"ERROR: %v\n", resp.Error)
			}

			if resp.Container == nil {
				log.Printf(op+"No container for running task %s\n", id)
				task.StateFailed(t)
				w.Db.Put(t.UUID.String(), t)
			}

			if resp.Container.State.Status == "exited" {
				log.Printf(op+"Container for task %s in non running state %s",
					id, resp.Container.State.Status)
				task.StateFailed(t)
				w.Db.Put(t.UUID.String(), t)
			}

			t.HostPorts = resp.Container.NetworkSettings.NetworkSettingsBase.Ports
			w.Db.Put(t.UUID.String(), t)
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
