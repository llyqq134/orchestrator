package manager

import (
	"encoding/json"
	"fmt"
	"log"
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
	}
}

