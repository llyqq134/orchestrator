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
	"orchestrator/pkg/store"
	"strings"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       queue.Queue
	TaskDb        store.Store
	EventDb       store.Store
	Workers       []string
	WorkerTaskMap map[string][]uuid.UUID
	TaskWorkerMap map[uuid.UUID]string
	LastWorker    int
	WorkerNodes   []*node.Node
	Scheduler     scheduler.Scheduler
}

func New(workers []string, schedulerType, dbType string) *Manager {
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
	case scheduler.EpvmScheduler:
		s = scheduler.New(scheduler.EpvmScheduler)
	default:
		s = scheduler.New(scheduler.RoundRobinScheduler)
	}

	m := Manager{
		Pending:       *queue.New(),
		Workers:       workers,
		WorkerTaskMap: workerTaskMap,
		TaskWorkerMap: taskWorkerMap,
		WorkerNodes:   nodes,
		Scheduler:     s,
	}

	var ts store.Store
	var es store.Store

	switch dbType {
	case "memory":
		ts = store.NewTaskStore()
		es = store.NewTaskEventStore()
	}

	m.TaskDb = ts
	m.EventDb = es

	return &m
}

func (m *Manager) SelectWorker(t task.Task) (*node.Node, error) {
	op := "[manager.SelectWorker]: "

	candidates := m.Scheduler.SelectCandidateNodes(t, m.WorkerNodes)
	if candidates == nil {
		msg := fmt.Sprintf("No candidate workers found for task %v", t.UUID)
		log.Println(op + msg)

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
		err := m.EventDb.Put(te.UUID.String(), &te)
		if err != nil {
			log.Printf(op+"Error attempting to store task event %s: %s\n", te.UUID.String(), err)
			return
		}

		log.Printf(op+"Pulled %v off pending queue\n", te)

		taskWorker, ok := m.TaskWorkerMap[te.Task.UUID]
		if ok {
			result, err := m.TaskDb.Get(te.Task.UUID.String())
			if err != nil {
				log.Printf(op+"Unable to schedule task: %s\n", err)
				return
			}

			persistedTask, ok := result.(*task.Task)
			if !ok {
				log.Println(op+"Cannot convert result to task.Task type: %v\n", err)
				return
			}

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

		log.Printf(op+"selected worker %v for task %v\n", w.Name, t.UUID)

		m.TaskWorkerMap[t.UUID] = w.Name
		task.StateScheduled(&t)

		data, err := json.Marshal(te)
		if err != nil {
			log.Printf(op+"Unable to marshal task object: %v\n", t)
		}

		url := fmt.Sprintf("http://%s/tasks", w.Name)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf(op+"Error connecting to %v: %v\n", w.Name, err)
			m.Pending.Enqueue(te)

			return
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			log.Printf(op+"Response error (%v): %s\n", resp.StatusCode, string(body))
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf(op+"Error reading response body: %v\n", err)
			return
		}

		if err := m.TaskDb.Put(t.UUID.String(), &t); err != nil {
			log.Printf(op+"Error storing task %v in db: %v\n", t.UUID, err)
			return
		}

		log.Printf(op+"Worker accepted task %v. Response: %s\n", t.UUID, string(body))
		log.Printf(op+"%#v\n", t)
	} else {
		log.Println(op + "No work in the queue")
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

	result, err := m.TaskDb.Get(taskId)
	if err != nil {
		log.Printf(op+"Error getting task %s from db: %v\n", taskId, err)
		return
	}

	t, ok := result.(*task.Task)
	if !ok {
		log.Println(op + "Cannot convert result to task.Task type\n")
		return
	}

	task.StateCompleted(t)
	if err := m.TaskDb.Put(taskId, t); err != nil {
		log.Printf(op+"Error updating task %s in db: %v\n", taskId, err)
	}

	log.Printf(op+"task %s has been stopped\n", taskId)
}

func (m *Manager) updateTasks() {
	op := "[manager.updateTasks]: "

	for _, worker := range m.Workers {
		log.Printf(op+"Checking worker %v for the task update\n", worker)
		url := fmt.Sprintf("http://%s/tasks", worker)

		resp, err := http.Get(url)
		if err != nil {
			log.Printf(op+"Error connecting to worker %v: %v\n", worker, err)
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf(op+"Error sengind request: %v\n", err)
		}

		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task

		if err := d.Decode(&tasks); err != nil {
			log.Printf(op+"Error unmarshalling tasks: %s\n", err.Error())
		}

		for _, t := range tasks {
			log.Printf("Attemting to update the task: %v\n", t.UUID)

			result, err := m.TaskDb.Get(t.UUID.String())
			if err != nil {
				log.Printf(op+"Task with UUID %v not found", t.UUID)
				continue
			}

			taskPersisted, ok := result.(*task.Task)
			if !ok {
				log.Printf(op+"cannot convert result %v to task.Task type\n", result)
				continue
			}

			if taskPersisted.State != t.State {
				taskPersisted.State = t.State
			}

			taskPersisted.StartTime = t.StartTime
			taskPersisted.FinishTime = t.FinishTime
			taskPersisted.ContainerID = t.ContainerID
			taskPersisted.HostPorts = t.HostPorts

			m.TaskDb.Put(taskPersisted.UUID.String(), taskPersisted)
		}
	}
}

func (m *Manager) UpdateTasks() {
	op := "[manager.UpdateTasks]: "

	for {
		log.Println(op + "Checking for task updates from workers")

		m.updateTasks()
		log.Println(op + "Task updates completed")

		timeToSleep := 10
		log.Printf(op+"Sleeping for %v seconds", timeToSleep)

		time.Sleep(time.Duration(timeToSleep) * time.Second)
	}
}

func (m *Manager) AddTask(te task.Event) {
	m.Pending.Enqueue(te)
}

func (m *Manager) GetTasks() []*task.Task {
	op := "[manager.GetTasks]: "

	/*
		tasks := []*task.Task{}
		for _, w := range m.Workers {
			url := fmt.Sprintf("http://%v/tasks", w)
			resp, err := http.Get(url)
			if err != nil {
				log.Printf(op+"Error connecting to %v: %v", w, err)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				log.Printf(op+"Response error (%v): %s\n", resp.StatusCode, string(body))
				continue
			}

			var workerTasks []*task.Task
			if err := json.NewDecoder(resp.Body).Decode(&workerTasks); err != nil {
				log.Printf(op+"Error decoding response body: %v\n", err)
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			tasks = append(tasks, workerTasks...)
		}

		return tasks
	*/

	taskList, err := m.TaskDb.List()
	if err != nil {
		log.Printf(op+"Error getting list of tasks: %v\n", err)
		return nil
	}

	return taskList.([]*task.Task)
}

func (m *Manager) ProcessTasks() {
	op := "[manager.ProcessTasks]: "
	for {
		log.Println(op + "Processing any tasks in the queue")

		m.SendWork()
		timeToSleep := 10
		log.Printf(op+"Sleeping for %v seconds", timeToSleep)

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
	m.TaskDb.Put(t.UUID.String(), t)

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
		log.Printf(op+"Error decoding response body: %v\n", err.Error())
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
