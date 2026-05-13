package store

import (
	"encoding/json"
	"fmt"
	"log"
	"orchestrator/pkg/resources/task"
	"os"

	"github.com/boltdb/bolt"
)

type TaskStore struct {
	Db       *bolt.DB
	DbFile   string
	FileMode os.FileMode
	Bucket   string
}

func NewTaskStore(file string, mode os.FileMode, bucket string) (*TaskStore, error) {
	op := "[store.NewTaskStore]: "

	db, err := bolt.Open(file, mode, nil)
	if err != nil {
		log.Printf(op+"Unable to open %v: %v", file, err)
		return nil, fmt.Errorf("Unable to open %v", file)
	}

	t := TaskStore{
		DbFile:   file,
		FileMode: mode,
		Db:       db,
		Bucket:   bucket,
	}

	if err = t.CreateBucket(); err != nil {
		log.Println(op + "Bucket already exists, will use it instead of creating new one")
	}

	return &t, nil
}

func (t *TaskStore) Close() {
	t.Db.Close()
}

func (t *TaskStore) Count() (int, error) {
	taskCount := 0

	err := t.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tasks"))
		b.ForEach(func(k, v []byte) error {
			taskCount++

			return nil
		})
		return nil
	})

	if err != nil {
		return -1, err
	}

	return taskCount, nil
}

func (t *TaskStore) CreateBucket() error {
	op := "[taskStore.CreateBucket]: "

	return t.Db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucket([]byte(t.Bucket)); err != nil {
			return fmt.Errorf(op+"Create bucket %s: %s", t.Bucket, err)
		}

		return nil
	})
}

func (t *TaskStore) Put(key string, value any) error {
	return t.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(t.Bucket))

		buf, err := json.Marshal(value.(*task.Task))
		if err != nil {
			return err
		}

		if err = b.Put([]byte(key), buf); err != nil {
			return err
		}

		return nil
	})
}

func (t *TaskStore) Get(key string) (any, error) {
	var task task.Task

	err := t.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(t.Bucket))
		t := b.Get([]byte(key))
		if t == nil {
			return fmt.Errorf("task %v not found", key)
		}

		if err := json.Unmarshal(t, &task); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &task, nil
}

func (t *TaskStore) List() (any, error) {
	var tasks []*task.Task

	err := t.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(t.Bucket))
		b.ForEach(func(k, v []byte) error {
			var task task.Task

			if err := json.Unmarshal(v, &task); err != nil {
				return err
			}
			tasks = append(tasks, &task)

			return nil
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return tasks, nil
}
