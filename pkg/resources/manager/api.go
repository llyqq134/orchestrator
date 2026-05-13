package manager

import (
	"encoding/json"
	"fmt"
	"log"
	"orchestrator/pkg/resources/task"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	healthURL = "/health" //GET
)

const (
	mainURL = "/tasks"

	getTasksURL              = ""       //GET
	getTasksURLTrailingSlash = "/"      //GET
	createTaskURL            = ""       //POST
	crateTaskURLTrailingSlsh = "/"      //POST
	deleteTaskURL            = "/:UUID" //DELETE
)

type Api struct {
	Host    string
	Port    string
	Manager *Manager
	Router  *gin.Engine
}

func (a *Api) Register() {
	tasks := a.Router.Group(mainURL)
	{
		tasks.GET(getTasksURL, a.GetTasksHandler)
		tasks.GET(getTasksURLTrailingSlash, a.GetTasksHandler)

		tasks.POST(createTaskURL, a.CreateTaskHandler)
		tasks.POST(crateTaskURLTrailingSlsh, a.CreateTaskHandler)

		tasks.DELETE(deleteTaskURL, a.DeleteTaskHandler)
	}
	a.Router.GET(healthURL, a.GetHealth)
}

func (a *Api) GetHealth(c *gin.Context) {
	c.JSON(200, nil)
}

func (a *Api) GetTasksHandler(c *gin.Context) {
	c.JSON(200, a.Manager.GetTasks())
}

func hasTaskPayload(t task.Task) bool {
	return t.UUID != uuid.Nil ||
		t.Name != "" ||
		t.Image != "" ||
		t.CPU != 0 ||
		t.Memory != 0 ||
		t.Disk != 0 ||
		len(t.ExposedPorts) != 0 ||
		len(t.PortBindings) != 0 ||
		t.RestartPolicy != ""
}

func (a *Api) CreateTaskHandler(c *gin.Context) {
	op := "[manager.CreteTaskHandler]: "

	te := task.Event{}
	flatTask := task.Task{}

	body, err := c.GetRawData()
	if err != nil {
		msg := fmt.Sprintf("Error reading body: %v", err)
		log.Println(op + msg)

		c.JSON(400, gin.H{
			"statusCode": 400,
			"Message":    msg,
		})

		return
	}

	if err := json.Unmarshal(body, &te); err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v", err)
		log.Println(op + msg)

		c.JSON(400, gin.H{
			"statusCode": 400,
			"Message":    msg,
		})

		return
	}

	if err := json.Unmarshal(body, &flatTask); err != nil && !hasTaskPayload(te.Task) && hasTaskPayload(flatTask) {
		te.Task = flatTask
	}

	te.UUID = uuid.New()
	te.Task.UUID = te.UUID
	a.Manager.AddTask(te)
	log.Printf(op+"Added task %v\n", te.UUID)

	c.JSON(201, gin.H{
		"Message": "Task created",
		"Id":      te.UUID,
	})
}

func (a *Api) DeleteTaskHandler(c *gin.Context) {
	op := "[manager.DeleteTaskHandler]: "

	strID := c.Param("UUID")

	if strID == "" {
		log.Println(op + "No task UUID passed in request")
		c.JSON(400, gin.H{
			"Message": "Bad request",
		})

		return
	}

	taskUUID, err := uuid.Parse(strID)
	if err != nil {
		c.JSON(503, gin.H{
			"Message": "Error parsing uuid",
		})

		return
	}

	result, err := a.Manager.TaskDb.Get(taskUUID.String())
	if err != nil {
		log.Printf(op+"No task with UUID: %v\n", taskUUID)
		c.JSON(404, gin.H{
			"statusCode": 404,
			"Message":    "No task with this UUID",
		})
	}

	taskToDelete, ok := result.(*task.Task)
	if !ok {
		log.Printf(op+"Cannot convert result to task.Task type: %v\n", err)
	}

	te := task.Event{
		UUID:      uuid.New(),
		State:     task.Completed,
		Timestamp: time.Now().UTC(),
	}

	taskCopy := *taskToDelete
	task.StateCompleted(&taskCopy)
	te.Task = taskCopy
	a.Manager.AddTask(te)

	log.Printf(op+"Added task event %v to delete task %v\n", te.UUID, taskToDelete.UUID)

	c.JSON(204, gin.H{})
}
