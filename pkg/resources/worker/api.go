package worker

import (
	"log"
	"orchestrator/pkg/resources/task"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Api struct {
	Host string
	Port string
	Worker *Worker
	Router *gin.Engine
}

const (
	startTaskURL = 		"/start" 						// POST
	getAllTasksURL = 	"/tasks" 						// GET
	getTaskByIdURL = 	"/tasks/:UUID" 			// GET
	deleteTaskURL = 	"/delete/:UUID" 		// DELETE
)

func (a *Api) Register () {
	a.Router.POST(startTaskURL, a.StartTaskHandler)
	a.Router.GET(getAllTasksURL, a.GetAllTasksHandler)
	a.Router.GET(getTaskByIdURL, a.GetTaskByIdHandler)
	a.Router.DELETE(deleteTaskURL, a.StopTaskHandler)
}

func (a *Api) StartTaskHandler(c *gin.Context) {
	te := task.Event{}

	if err := c.BindJSON(&te); err != nil {
		log.Printf("Error binding a task: %v\n", err.Error())
		c.JSON(400, gin.H {
			"message": "Bad request",
			"error": err.Error(),
		})

		return
	}

	a.Worker.AddTask(te.Task)
	log.Printf("Added task %v\n", te.Task.UUID)

	c.JSON(201, gin.H {
		"message": "Task was created",
	})
}

func (a *Api) GetAllTasksHandler(c *gin.Context) {
	c.JSON(200, a.Worker.GetTasks())
}

func (a *Api) GetTaskByIdHandler(c *gin.Context) {
	strID := c.Param("UUID")

	if strID == "" {
		log.Println("No task passed in request")

		c.JSON(400, gin.H {
			"message": "No task passed in request",
		})
	}

	taskId, err := uuid.Parse(strID)
	if err != nil {
		log.Printf("Error parsing uuid: %v\n", err)
		c.JSON(400, gin.H{
			"message": "Bad request",
			"error": err.Error,
		})

		return
	}

	if 	_, ok := a.Worker.Db[taskId]; !ok {
		log.Printf("No task with id %v found\n", taskId)
		c.JSON(404, gin.H {
			"message": "Task wasn't found",
		})
		return
	}

	c.JSON(200, a.Worker.Db[taskId])
}

func (a *Api) StopTaskHandler(c *gin.Context) {
	strID := c.Param("UUID")

	if strID == "" {
		log.Println("No taskId passed in request")
		c.JSON(400, gin.H {
			"message": "No taskID passed in request",
		})

		return
	}

	taskId, err := uuid.Parse(strID)
	if err != nil {
		log.Printf("error parsing uuid: %v\n", err)
		c.JSON(400, gin.H {
			"message": "error parsing uuid",
			"error": err.Error,
		})

		return
	}
	
	if _, ok := a.Worker.Db[taskId]; !ok {
		log.Printf("No task with id %v found\n", taskId)

		c.JSON(404, gin.H {
			"message": "task wasn't found",
		})

		return
	}

	taskToStop := a.Worker.Db[taskId]
	taskCopy := *taskToStop
	task.StateCompleted(taskCopy)
	a.Worker.StopTask(taskCopy)

	log.Printf("Added task %v to stop container %v\n", taskToStop.UUID, taskToStop.ContainerID)

	c.JSON(204, gin.H {
		"message": "task was deleted",
	})
}
