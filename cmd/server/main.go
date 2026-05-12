package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"orchestrator/config"
	"orchestrator/pkg/resources/manager"
	"orchestrator/pkg/resources/scheduler"
	"orchestrator/pkg/resources/task"
	"orchestrator/pkg/resources/worker"

	"github.com/gin-gonic/gin"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
)

const (
	workersCount = 3
)

func main() {
	managerRouter := gin.Default()

	var cfg config.Config

	if err := cleanenv.ReadConfig("./../../config/server.yaml", &cfg); err != nil {
		panic(err)
	}

	basePort, err := strconv.Atoi(cfg.Worker.Port)
	if err != nil {
		panic(fmt.Errorf("invalid worker port: %w", err))
	}

	workers := make([]*worker.Worker, workersCount)
	workerAddrs := make([]string, workersCount)

	for i := range workersCount {
		w := &worker.Worker{
			Queue: *queue.New(),
			Db:    make(map[uuid.UUID]*task.Task),
		}
		workers[i] = w

		addr := fmt.Sprintf("%s:%d", cfg.Worker.Host, basePort+i)
		workerAddrs[i] = addr

		r := gin.Default()
		wapi := worker.Api{
			Host:   cfg.Worker.Host,
			Port:   strconv.Itoa(basePort + i),
			Worker: w,
			Router: r,
		}
		wapi.Register()

		go func() {
			if err := r.Run(addr); err != nil {
				panic(err)
			}
		}()
	}

	for _, addr := range workerAddrs {
		waitForServer("http://" + addr + "/health")
	}

	for _, w := range workers {
		go w.RunTasks()
		go w.CollectStats()
	}

	m := manager.New(workerAddrs, scheduler.EpvmScheduler)

	managerApi := manager.Api{
		Host:    cfg.Manager.Host,
		Port:    cfg.Manager.Port,
		Manager: m,
		Router:  managerRouter,
	}
	managerApi.Register()

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoTaskHealthCheck()

	managerAddr := fmt.Sprintf("%s:%s", cfg.Manager.Host, cfg.Manager.Port)
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
