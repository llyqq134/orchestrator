package store

import (
	"fmt"
	"orchestrator/pkg/resources/task"
)

type InMemoryTaskStore struct {
	Db map[string]*task.Task
}

func NewTaskStore() *InMemoryTaskStore {
	return &InMemoryTaskStore{
		Db: make(map[string]*task.Task),
	}
}

func (s *InMemoryTaskStore) Put(key string, value any) error {
	op := "[InMemoryTaskStore.Put]: "

	t, ok := value.(*task.Task)
	if !ok {
		return fmt.Errorf(op+"Value %v is not a task.Task type", value)
	}

	s.Db[key] = t

	return nil
}

func (s *InMemoryTaskStore) Get(key string) (any, error) {
	op := "[InMemoryTaskStore.Get]: "

	t, ok := s.Db[key]
	if !ok {
		return nil, fmt.Errorf(op+"Task with key %s does not exist", key)
	}

	return t, nil
}

func (s *InMemoryTaskStore) List() (any, error) {
	var tasks []*task.Task
	for _, t := range s.Db {
		tasks = append(tasks, t)
	}

	return tasks, nil
}

func (s *InMemoryTaskStore) Count() (int, error) {
	return len(s.Db), nil
}
