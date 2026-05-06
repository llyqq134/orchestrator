package manager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"orchestrator/pkg/resources/task"

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
	LastWorker int 
}

func (m *Manager) SelectWorker() string {
	var newWorker int 

	if m.LastWorker < len(m.Workers) {
		m.LastWorker++
		newWorker = m.LastWorker
	} else {
		newWorker = 0
		m.LastWorker = 0
	}

	return m.Workers[newWorker]
}

func (m *Manager) UpdateTasks() {
	fmt.Println("Update tasks")
}

func (m *Manager) SendWork() {
	if m.Pending.Len() > 0 {
		chosenWorker := m.SelectWorker()
		
		e := m.Pending.Dequeue()
		taskEvent := e.(task.Event)
		t := taskEvent.Task

		log.Printf("Pulled %v off pending queue\n", t)

		m.EventDb[t.UUID] = &taskEvent
		m.WorkerTaskMap[chosenWorker] = append(m.WorkerTaskMap[chosenWorker], taskEvent.Task.UUID)
		m.TaskWorkerMap[t.UUID] = chosenWorker

		t.State = task.Scheduled
		m.TaskDb[t.UUID] = &t 

		data, err := json.Marshal(taskEvent)
		if err != nil {
			log.Printf("unable to marshal task object: %v\n", t)
		}

		url := fmt.Sprintf("http://%s/tasks", chosenWorker)

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			fmt.Sprintf("Error connecting to %v: %v\n", chosenWorker, err)
			m.Pending.Enqueue(taskEvent)

			return
		}

		d := json.NewDecoder(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			if err := d.Decode(&e); err != nil {
				fmt.Printf("Error decoding respose: %v\n", err.Error())
				return
			}
			log.Printf("Response error (%v)\n", resp.StatusCode)
			return
		}

		t = task.Task{}
		if err := d.Decode(&t); err != nil {
			fmt.Printf("Error decoding response: %v\n", err.Error())
			return 
		}
		log.Printf("%#v\n", t)
	} else {
		log.Println("No work in the queue")
	}
}

func (m *Manager)UpdateTask() {
	for _, worker := range m.Workers {
		log.Printf("Checking worker %v for the task update\n", worker)
		url := fmt.Sprintf("http://%s/tasks", worker)

		resp, err := http.Get(url) 
		if err != nil {
			log.Printf("Error connecting to worker %v: %v\n", worker, err)
		}	

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error sengind request: %v\n", err)
		}

		d := json.NewDecoder(resp.Body)
		var tasks []*task.Task

		if err := d.Decode(&tasks); err != nil {
			log.Printf("Error unmarshalling tasks: %s\n", err.Error())
		}

		for _, t := range tasks {
			log.Printf("Attemting to update the task: %v\n", t.UUID)
			
			_, ok := m.TaskDb[t.UUID]
			if !ok {
				log.Printf("Task with UUID %v not found", t.UUID)
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

