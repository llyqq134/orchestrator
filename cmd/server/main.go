package main

import (
	"fmt"
	"log"
	"orchestrator/config"
	"orchestrator/pkg/resources/task"
	"orchestrator/pkg/resources/worker"
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
	router.Use()
	router.Run(fmt.Sprintf("%v:%v", cfg.Host, cfg.Port)	)
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
