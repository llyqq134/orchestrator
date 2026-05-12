package manager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"orchestrator/pkg/resources/node"
	"orchestrator/pkg/resources/scheduler"
	"orchestrator/pkg/resources/task"
	"strings"
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
	Scheduler     scheduler.Scheduler
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

	var s scheduler.Scheduler
	switch schedulerType {
	case scheduler.RoundRobinScheduler:
		s = scheduler.New(scheduler.RoundRobinScheduler)
	default:
		s = scheduler.New(scheduler.RoundRobinScheduler)
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

func (m *Manager) SelectWorker(t task.Task) (*node.Node, error) {
	candidates := m.Scheduler.SelectCandidateNodes(t, m.WorkerNodes)
	if candidates == nil || len(candidates) == 0 {
		msg := fmt.Sprintf("No candidate workers found for task %v", t.UUID)
		log.Println("[manager.SelectWorker]: " + msg)

		err := errors.New(msg)
		return nil, err
	}

	scores := m.Scheduler.Score(t, candidates)
	selectNode := m.Scheduler.Pick(scores, candidates)

	return selectNode, nil
}

func (m *Manager) SendWork() {
	op := "[manager.SendWork]: "

	if m.Pending.Len() > 0 {
		e := m.Pending.Dequeue()
		te := e.(task.Event)
		m.EventDb[te.UUID] = &te

		log.Printf(op+"Pulled %v off pending queue\n", te)

		taskWorker, ok := m.TaskWorkerMap[te.Task.UUID]
		if ok {
			persistedTask := m.TaskDb[te.Task.UUID]
			if te.State == task.Completed && task.ValidateTransition(persistedTask.State, te.State) {
				m.stopTask(taskWorker, te.Task.UUID.String())
				return
			}

			log.Printf(op+
				"Invalid request: existing task %v is in state %v and cannot transition to the completed%v\n",
				te.Task.UUID, taskWorker, te.State)

			return
		}

		t := te.Task
		w, err := m.SelectWorker(t)
		if err != nil {
			log.Printf(op+"Error selecting worker for task %v: %v\n", t.UUID, err)
			return
		}

		log.Printf(op+"slected worker %v for task %v\n", w.Name, t.UUID)

		m.TaskWorkerMap[t.UUID] = w.Name
		task.StateScheduled(&t)

		data, err := json.Marshal(te)
		if err != nil {
			log.Printf(op+"%v: Unable to marshal task object: %v\n", op, t)
		}

		url := fmt.Sprintf("http://%s/tasks", w.Name)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf(op+"%v: Error connecting to %v: %v\n", op, w.Name, err)
			m.Pending.Enqueue(te)

			return
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			log.Printf(op+"%v: Response error (%v): %s\n", op, resp.StatusCode, string(body))
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf(op+"%v: Error reading response body: %v\n", op, err)
			return
		}

		log.Printf(op+"%v: Worker accepted task %v. Response: %s\n", op, t.UUID, string(body))
		log.Printf(op+"%#v\n", t)
	} else {
		log.Println("No work in the queue")
	}
}

func (m *Manager) stopTask(worker string, taskId string) {
	op := "[manager.stopTask]: "

	client := &http.Client{}
	url := fmt.Sprintf("http://%s/tasks/%s", worker, taskId)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		log.Printf(op+"Error creating request: %v\n", err)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf(op+"Error connecting to worker at %v: %v\n", worker, err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		log.Printf(op+"Error sending request: %v\n", err)
		return
	}

	log.Printf(op+"task %s has been scheduled to be stopped\n", taskId)
}

func (m *Manager) updateTasks() {
	op := "[manager.updateTasks]: "

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

func (m *Manager) checkTaskHealth(t task.Task) error {
	op := "[manager.checkTaskHealth]: "
	log.Printf(op+"Calling health check for task %s: %s\n", t.UUID, t.HealthCheck)

	w := m.TaskWorkerMap[t.UUID]
	hostPort := t.GetHostPort(t.HostPorts)
	if hostPort == nil {
		log.Printf(op+"Have not collected task %s host port yet. Skipping.", t.UUID)
		return nil
	}

	worker := strings.Split(w, ":")
	url := fmt.Sprintf("http://%s:%s%s", worker[0], *hostPort, t.HealthCheck)

	log.Printf(op+"Calling health check for task %s: %s\n", t.UUID, url)

	resp, err := http.Get(url)
	if err != nil {
		msg := fmt.Sprintf("Error connecting to health check %s", url)
		log.Println(op + msg)

		return errors.New(msg)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Error health check for task %s did not return 200\n", t.UUID)
		log.Println(op + msg)

		return errors.New(msg)
	}

	log.Printf(op+"Task %s Health check response: %v\n", t.UUID, resp.StatusCode)

	return nil
}

func (m *Manager) doTaskHealthCheck() {
	for _, t := range m.GetTasks() {
		if t.State == task.Running && t.RestartCount < 3 {
			if err := m.checkTaskHealth(*t); err != nil && t.RestartCount < 3 {
				m.restartTask(t)
			}
		} else if t.State == task.Failed && t.RestartCount < 3 {
			m.restartTask(t)
		}
	}
}

func (m *Manager) restartTask(t *task.Task) {
	op := "[manager.restartTask]: "

	w := m.TaskWorkerMap[t.UUID]
	t.State = task.Scheduled
	t.RestartCount++
	m.TaskDb[t.UUID] = t

	te := task.Event{
		UUID:      uuid.New(),
		State:     task.Running,
		Timestamp: time.Now(),
		Task:      *t,
	}

	data, err := json.Marshal(te)
	if err != nil {
		log.Printf(op+"Unable to marshal task object: %v", t)
		return
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		log.Printf(op+"Error connecting to %v: %v\n", w, err)
		m.Pending.Enqueue(t)
	}

	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		log.Printf(op+"Response error (%v): %s\n", resp.StatusCode, string(body))
		return
	}

	newTask := task.Task{}
	if err = d.Decode(&newTask); err != nil {
		log.Printf("%v: Error decoding response body: %v\n", op, err.Error())
		return
	}

	log.Printf(op+"%#v\n", t)
}

func (m *Manager) DoTaskHealthCheck() {
	op := "[manager.DoTaskHealthCheck]: "

	for {
		log.Println(op + "Performing task health check")
		m.doTaskHealthCheck()

		log.Println(op + "Task health checks completed")

		timeToSleep := 45
		log.Printf(op+"Sleeping for %v seconds\n", timeToSleep)
		time.Sleep(time.Duration(timeToSleep) * time.Second)
	}
}
