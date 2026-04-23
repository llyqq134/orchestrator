package manager

import (
	"fmt"
	"orchestrator/pkg/resources/task"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
	Pending       queue.Queue
	TaskDb        map[uuid.UUID][]*task.Task
	EventDb       map[uuid.UUID][]*task.Event
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
	fmt.Println("Send work")
}
