package store

import (
	"encoding/json"
	"fmt"
	"log"
	"orchestrator/pkg/resources/task"
	"os"

	"github.com/boltdb/bolt"
)

type EventStore struct {
	DbFile   string
	FileMode os.FileMode
	Db       *bolt.DB
	Bucket   string
}

func NewEventStore(file string, mode os.FileMode, bucket string) (*EventStore, error) {
	op := "[store.NewEventStore]: "

	db, err := bolt.Open(file, mode, nil)
	if err != nil {
		return nil, fmt.Errorf(op+"Unable to open %v: %v", file, err)
	}

	e := EventStore{
		DbFile:   file,
		FileMode: mode,
		Db:       db,
		Bucket:   bucket,
	}

	if err := e.CreateBucket(); err != nil {
		log.Println(op + "Bucket already exists, will use it instead of creating new one")
	}

	return &e, nil
}

func (e *EventStore) Close() {
	e.Db.Close()
}

func (e *EventStore) CreateBucket() error {
	op := "[eventStore.CreateBucket]: "

	return e.Db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucket([]byte(e.Bucket)); err != nil {
			return fmt.Errorf(op+"Create bucket %s: %s", e.Bucket, err)
		}

		return nil
	})
}

func (e *EventStore) Count() (int, error) {
	eventCount := 0

	err := e.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))
		b.ForEach(func(k, v []byte) error {
			eventCount++

			return nil
		})
		return nil
	})

	if err != nil {
		return -1, err
	}

	return eventCount, nil
}

func (e *EventStore) Put(key string, value any) error {
	return e.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))

		buf, err := json.Marshal(value.(*task.Event))
		if err != nil {
			return err
		}

		if err = b.Put([]byte(key), buf); err != nil {
			return err
		}

		return nil
	})
}

func (e *EventStore) List() (any, error) {
	var events []*task.Event

	err := e.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))
		b.ForEach(func(k, v []byte) error {
			var event task.Event
			if err := json.Unmarshal(v, &event); err != nil {
				return err
			}
			events = append(events, &event)
			return nil
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return events, nil
}

func (e *EventStore) Get(key string) (any, error) {
	op := "[eventStore.Get]: "

	var events task.Event

	err := e.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(e.Bucket))
		t := b.Get([]byte(key))
		if t == nil {
			return fmt.Errorf(op+"Event %v not found", key)
		}

		if err := json.Unmarshal(t, &events); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return events, nil
}
