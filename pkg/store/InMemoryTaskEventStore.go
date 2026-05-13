package store

import (
	"fmt"
	"orchestrator/pkg/resources/task"
)

type InMemoryTaskEventStore struct {
	Db map[string]*task.Event
}

func NewInMemoryTaskEventStore() *InMemoryTaskEventStore {
	return &InMemoryTaskEventStore{
		Db: make(map[string]*task.Event),
	}
}

func (s *InMemoryTaskEventStore) Put(key string, value any) error {
	op := "[InMemoryTaskEventStore.Put]: "

	e, ok := value.(*task.Event)
	if !ok {
		return fmt.Errorf(op+"Value &v is not a task.Event type", value)
	}

	s.Db[key] = e

	return nil
}

func (s *InMemoryTaskEventStore) Get(key string) (any, error) {
	op := "[InMemoryTaskEventStore.Get]: "

	e, ok := s.Db[key]
	if !ok {
		return nil, fmt.Errorf(op+"Task event with key %s does not exist", key)
	}

	return e, nil
}

func (s *InMemoryTaskEventStore) List() (any, error) {
	var events []*task.Event
	for _, e := range s.Db {
		events = append(events, e)
	}

	return events, nil
}

func (s *InMemoryTaskEventStore) Count() (int, error) {
	return len(s.Db), nil
}
