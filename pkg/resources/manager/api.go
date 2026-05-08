package manager

import (
	"fmt"
	"log"
	"orchestrator/pkg/resources/task"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	mainURL = "/tasks"

	getTasksURL = "/"						//GET
	createTaskURL = "/"					//POST 
	deleteTaskURL = "/:UUID"		//DELETE 
)

type Api struct {
	Host string 
	Port string 
	Manager *Manager
	Router *gin.Engine
} 

func (a *Api)Register() {
	tasks := a.Router.Group(mainURL)	 
	{
		tasks.GET(getTasksURL, a.GetTasksHandler)
		tasks.POST(createTaskURL, a.CreateTaskHandler)
		tasks.DELETE(deleteTaskURL, a.DeleteTaskHandler)
	}
}

func (a *Api)GetTasksHandler(c *gin.Context) {
	c.JSON(200, a.Manager.GetTasks())
}

func (a *Api)CreateTaskHandler(c *gin.Context) {
	var te task.Event

	if err := c.ShouldBindJSON(&te); err != nil {
		msg := fmt.Sprintf("Error unmarshalling body: %v", err)
		log.Printf(msg)

		c.JSON(400, gin.H {
			"statusCode": 400,
			"Message": msg,
		})																							

		return				
	}

	a.Manager.AddTask(te)
	log.Printf("Added task %v\n", te.Task.UUID)

	c.JSON(201, gin.H {
		"Message": "Task created",
	})
}

func (a *Api)DeleteTaskHandler(c *gin.Context) {
	strID := c.Param("UUID")

	if strID == "" {
		log.Printf("No task UUID passed in request\n")
		c.JSON(400, gin.H {
			"Message": "Bad request",
		})

		return 
	}

	taskUUID, err := uuid.Parse(strID)	
	if err != nil {
		c.JSON(503, gin.H {
			"Message": "Error parsing uuid",
		})

		return
	}

}

