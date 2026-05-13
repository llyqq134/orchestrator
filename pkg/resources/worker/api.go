package worker

import (
	"log"
	"orchestrator/pkg/resources/task"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Api struct {
	Host   string
	Port   string
	Worker *Worker
	Router *gin.Engine
}

const (
	healthURL = "/health" // GET
)

const (
	mainTaskUrl = "/tasks"

	startTaskURL                = ""       // POST
	startTaskURLTrailingSlash   = "/"      // POST
	getAllTasksURL              = ""       // GET
	getAllTasksURLTrailingSlash = "/"      // Get
	getTaskByIdURL              = "/:UUID" // GET
	deleteTaskURL               = "/:UUID" // DELETE
)

const (
	mainStatUrl = "/stats"

	statsURL = "" // GET
)

func (a *Api) Register() {
	a.Router.GET(healthURL, a.GetHealth)
	tasks := a.Router.Group(mainTaskUrl)
	{
		tasks.POST(startTaskURL, a.StartTaskHandler)
		tasks.POST(startTaskURLTrailingSlash, a.StartTaskHandler)

		tasks.GET(getAllTasksURL, a.GetAllTasksHandler)
		tasks.GET(getAllTasksURLTrailingSlash, a.GetAllTasksHandler)

		tasks.GET(getTaskByIdURL, a.GetTaskByIdHandler)

		tasks.DELETE(deleteTaskURL, a.StopTaskHandler)
	}

	stats := a.Router.Group(mainStatUrl)
	{
		stats.GET(statsURL, a.GetStatsHandler)
	}
}

func (a *Api) GetHealth(c *gin.Context) {
	c.JSON(200, nil)
}

func (a *Api) StartTaskHandler(c *gin.Context) {
	op := "[worker.StartTaskHandler]: "

	te := task.Event{}

	if err := c.BindJSON(&te); err != nil {
		log.Printf(op+"Error binding a task: %v\n", err.Error())
		c.JSON(400, gin.H{
			"message": "Bad request",
			"error":   err.Error(),
		})

		return
	}

	if te.UUID == uuid.Nil {
		te.UUID = uuid.New()
	}

	if te.Task.UUID == uuid.Nil {
		te.Task.UUID = te.UUID
	}

	a.Worker.AddTask(te.Task)
	log.Printf(op+"Added task %v\n", te.Task.UUID)

	c.JSON(201, gin.H{
		"message": "Task was created",
		"id":      te.UUID,
	})
}

func (a *Api) GetAllTasksHandler(c *gin.Context) {
	c.JSON(200, a.Worker.GetTasks())
}

func (a *Api) GetTaskByIdHandler(c *gin.Context) {
	op := "[worker.GetTaskByIdHandler]: "

	strID := c.Param("UUID")

	if strID == "" {
		log.Println(op + "No task passed in request")

		c.JSON(400, gin.H{
			"message": "No task passed in request",
		})
	}

	taskId, err := uuid.Parse(strID)
	if err != nil {
		log.Printf(op+"Error parsing uuid: %v\n", err)
		c.JSON(400, gin.H{
			"message": "Bad request",
			"error":   err.Error,
		})

		return
	}

	task, err := a.Worker.Db.Get(taskId.String())
	if err != nil {
		log.Printf(op+"No task with id %v found\n", taskId)
		c.JSON(404, gin.H{
			"message": "Task wasn't found",
		})
		return
	}

	c.JSON(200, task)
}

func (a *Api) StopTaskHandler(c *gin.Context) {
	op := "[worker.StopTaskHandler]: "

	strID := c.Param("UUID")

	if strID == "" {
		log.Println(op + "No taskId passed in request")
		c.JSON(400, gin.H{
			"message": "No taskID passed in request",
		})

		return
	}

	taskId, err := uuid.Parse(strID)
	if err != nil {
		log.Printf(op+"error parsing uuid: %v\n", err)
		c.JSON(400, gin.H{
			"message": "error parsing uuid",
			"error":   err.Error,
		})

		return
	}

	taskToStop, err := a.Worker.Db.Get(taskId.String())
	if err != nil {
		log.Printf(op+"No task with id %v found\n", taskId)

		c.JSON(404, gin.H{
			"message": "task wasn't found",
		})

		return
	}

	taskCopy := *taskToStop.(*task.Task)

	task.StateCompleted(&taskCopy)
	a.Worker.StopTask(taskCopy)

	log.Printf(op+"Added task %v to stop container %v\n", taskCopy.UUID.String(), taskCopy.ContainerID)

	c.JSON(204, gin.H{
		"message": "task was deleted",
	})
}

func (a *Api) GetStatsHandler(c *gin.Context) {
	c.JSON(200, a.Worker.Stats)
}
