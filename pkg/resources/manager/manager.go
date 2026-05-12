package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"orchestrator/pkg/resources/node"
	"orchestrator/pkg/resources/scheduler"
	"orchestrator/pkg/resources/task"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       queue.Queue
	TaskDb        map[uuid.UUID]*task.Task
	EventDb       map[uuid.UUID]*task.Event
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	LastWorker    int
	WorkerNodes   []*node.Node
	Scheduler     scheduler.SchedulerType
}

func New(workers []string, schedulerType string) *Manager {
	taskDB := make(map[uuid.UUID]*task.Task)
	eventDB := make(map[uuid.UUID]*task.Event)
	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)

	var nodes []*node.Node
	for worker := range workers {
		workerTaskMap[workers[worker]] = []uuid.UUID{}

		nAPI := fmt.Sprintf("http://%v", workers[worker])
		n := node.New(workers[worker], nAPI, "worker")
		nodes = append(nodes, n)
	}

	var s scheduler.SchedulerType
	switch schedulerType {
	case "roundRobin":
		s = &scheduler.SchedulerType.RoundRobin{Name: "roundRobin"}
	default:
		s = &scheduler.RoundRobin{Name: "roundRobin"}
	}

	return &Manager{
		Pending:       *queue.New(),
		TaskDb:        taskDB,
		EventDb:       eventDB,
		Workers:       workers,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
		WorkerNodes:   nodes,
		Scheduler:     s,
	}
}

func (m *Manager) SelectWorker() string {
	var newWorker int

	if m.LastWorker+1 < len(m.Workers) {
		newWorker = m.LastWorker + 1
		m.LastWorker++
	} else {
		newWorker = 0
		m.LastWorker = 0
	}

	return m.Workers[newWorker]
}

func (m *Manager) SendWork() {
	op := "manager.SendWork"

	if m.Pending.Len() > 0 {
		chosenWorker := m.SelectWorker()

		e := m.Pending.Dequeue()
		te := e.(task.Event)
		t := te.Task

		log.Printf("Pulled %v off pending queue\n", t)

		m.EventDb[te.UUID] = &te
		m.WorkerTaskMap[chosenWorker] = append(m.WorkerTaskMap[chosenWorker], te.Task.UUID)
		m.TaskWorkerMap[t.UUID] = chosenWorker

		task.StateScheduled(&t)
		m.TaskDb[t.UUID] = &t

		data, err := json.Marshal(te)
		if err != nil {
			log.Printf("%v: Unable to marshal task object: %v\n", op, t)
		}

		url := fmt.Sprintf("http://%s/tasks", chosenWorker)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Printf("%v: Error connecting to %v: %v\n", op, chosenWorker, err)
			m.Pending.Enqueue(te)

			return
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("%v: Response error (%v): %s\n", op, resp.StatusCode, string(body))
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("%v: Error reading response body: %v\n", op, err)
			return
		}

		log.Printf("%v: Worker accepted task %v. Response: %s\n", op, t.UUID, string(body))
		log.Printf("%#v\n", t)
	} else {
		log.Println("No work in the queue")
	}
}

func (m *Manager) updateTasks() {
	op := "manager.updateTasks"

	for _, worker := range m.Workers {
		log.Printf("Checking worker %v for the task update\n", worker)
		url := fmt.Sprintf("http://%s/tasks", worker)

		resp, err := http.Get(url)
		if err != nil {
			log.Printf("%v: Error connecting to worker %v: %v\n", op, worker, err)
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("%v: Error sengind request: %v\n", op, err)
		}

		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task

		if err := d.Decode(&tasks); err != nil {
			log.Printf("%v: Error unmarshalling tasks: %s\n", op, err.Error())
		}

		for _, t := range tasks {
			log.Printf("Attemting to update the task: %v\n", t.UUID)

			_, ok := m.TaskDb[t.UUID]
			if !ok {
				log.Printf("%v: Task with UUID %v not found", op, t.UUID)
				return
			}

			if m.TaskDb[t.UUID].State != t.State {
				m.TaskDb[t.UUID].State = t.State
			}

			m.TaskDb[t.UUID].StartTime = t.StartTime
			m.TaskDb[t.UUID].FinishTime = t.FinishTime
			m.TaskDb[t.UUID].ContainerID = t.ContainerID
		}
	}
}

func (m *Manager) UpdateTasks() {
	for {
		log.Println("Checking for task updates from workers")

		m.updateTasks()
		log.Println("Task updates completed")

		timeToSleep := 10
		log.Printf("Sleeping for %v seconds", timeToSleep)

		time.Sleep(time.Duration(timeToSleep) * time.Second)
	}
}

func (m *Manager) AddTask(te task.Event) {
	m.Pending.Enqueue(te)
}

func (m *Manager) GetTasks() []*task.Task {
	tasks := []*task.Task{}
	for _, t := range m.TaskDb {
		tasks = append(tasks, t)
	}

	return tasks
}

func (m *Manager) ProcessTasks() {
	for {
		log.Println("Processing any tasks in the queue")

		m.SendWork()
		timeToSleep := 10
		log.Printf("Sleeping for %v seconds", timeToSleep)

		time.Sleep(time.Duration(timeToSleep) * time.Second)
	}
}
