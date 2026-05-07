package main

import (
	"fmt"
	"log"
	"orchestrator/config"
	"orchestrator/pkg/resources/task"
	"orchestrator/pkg/resources/worker"
	"orchestrator/pkg/resources/manager"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
)

func main() {
	router := gin.Default()

	var cfg config.Server
	err := cleanenv.ReadConfig("../../config/server.yaml", &cfg)
	if err != nil {
		panic(err)
	}

	fmt.Println("Starting cube worker")

	w := worker.Worker {
		Queue: *queue.New(),
		Db: make(map[uuid.UUID]*task.Task),
	}

	api := worker.Api{Host: cfg.Host, Port: cfg.Port, Worker: &w, Router: router}
	api.Register()

	go RunTasks(&w)
	go w.CollectStats()

	go func() {
		router.Use()
  	router.Run(fmt.Sprintf("%v:%v", cfg.Host, cfg.Port))
	}()

	workers := []string{fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)}
	m := manager.New(workers)

	for i := range 3 {
		t := task.Task {
			UUID: uuid.New(),
			Name: fmt.Sprintf("test-container-%d", i),
			State: task.Scheduled,
			Image: "strm/helloworld-http",
		}

		te := task.Event {
			UUID: uuid.New(),
			State: task.Running,
			Task: t,
		}

		m.AddTask(te)
		m.SendWork()
	}

	go func() {
		for {
			fmt.Printf("[Manager] Updating task from %d workers\n", len(m.Workers))
			m.UpdateTasks()
			time.Sleep(15 * time.Second)
		}
	}()

	for {
		for _, t := range m.TaskDb {
			fmt.Printf("[Manager] Task:\n\tUUID: %v\n\tState: %v\n", t.UUID, t.State)
			time.Sleep(15 * time.Second)
		}
	}
}

func RunTasks(w *worker.Worker) {
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n", result.Error)
			} 
		} else {
			log.Println("No tasks to process currently")
		}

		log.Println("waiting for 10 sec")
		time.Sleep(time.Second * 10)
	}
}
