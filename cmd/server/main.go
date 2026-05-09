package main 

import (
	"fmt"
	"net/http"
	"time"

	"orchestrator/config"
	"orchestrator/pkg/resources/manager"
	"orchestrator/pkg/resources/task"
	"orchestrator/pkg/resources/worker"

	"github.com/gin-gonic/gin"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
)

func main() {
	workerRouter := gin.Default()
	managerRouter := gin.Default()

	var cfg config.Config

	err := cleanenv.ReadConfig("../../config/server.yaml", &cfg)
	if err != nil {
		panic(err)
	}

	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
	}

	workers := []string{
		fmt.Sprintf("%v:%v", cfg.Worker.Host, cfg.Worker.Port),
	}

	m := manager.New(workers)

	workerApi := worker.Api{
		Host:   cfg.Worker.Host,
		Port:   cfg.Worker.Port,
		Worker: &w,
		Router: workerRouter,
	}

	workerApi.Register()

	managerApi := manager.Api{
		Host:    cfg.Manager.Host,
		Port:    cfg.Manager.Port,
		Manager: m,
		Router:  managerRouter,
	}

	managerApi.Register()

	workerAddr := fmt.Sprintf(
		"%v:%v",
		cfg.Worker.Host,
		cfg.Worker.Port,
	)

	go func() {
		if err := workerRouter.Run(workerAddr); err != nil {
			panic(err)
		}
	}()

	waitForServer("http://" + workerAddr + "/health")

	go w.RunTasks()
	go w.CollectStats()
	go m.ProcessTasks()
	go m.UpdateTasks()

	managerAddr := fmt.Sprintf(
		"%v:%v",
		cfg.Manager.Host,
		cfg.Manager.Port,
	)

	if err := managerRouter.Run(managerAddr); err != nil {
		panic(err)
	}
}

func waitForServer(url string) {
	for {
		resp, err := http.Get(url)

		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}

		if resp != nil {
			resp.Body.Close()
		}

		time.Sleep(100 * time.Millisecond)
	}
}
