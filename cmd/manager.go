/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"orchestrator/pkg/resources/manager"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

// managerCmd represents the manager command
var managerCmd = &cobra.Command{
	Use:   "manager",
	Short: "Manager command to operate a cube manager node",
	Long: `The manager controls the orchestration system and is responsible for:
	- Accepting tasks from users
	- Scheduling tasks onto worker nodes
	- Rescheduling tasks in the event of a node failure
	- Periodically send a task to workers to get task updates`,
	Run: func(cmd *cobra.Command, args []string) {
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		workers, _ := cmd.Flags().GetStringSlice("workers")
		scheduler, _ := cmd.Flags().GetString("scheduler")
		dbType, _ := cmd.Flags().GetString("dbtype")
		dataDir, _ := cmd.Flags().GetString("datadir")

		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}

		log.Println("Starting manager")

		router := gin.Default()
		m := manager.New(workers, scheduler, dbType, dataDir)
		api := manager.Api{Host: host, Port: strconv.Itoa(port), Manager: m, Router: router}

		go m.ProcessTasks()
		go m.UpdateTasks()
		go m.DoTaskHealthCheck()

		log.Printf("Starting manager API on %s:%d\n", host, port)

		api.Register()
		if err := router.Run(fmt.Sprintf("%s:%d", host, port)); err != nil {
			log.Fatalf("Failed to start manager API: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(managerCmd)
	managerCmd.Flags().StringP("host", "H", "0.0.0.0", "Hostname or IP address")
	managerCmd.Flags().IntP("port", "p", 5555, "Port on which to listen")
	managerCmd.Flags().StringSliceP("workers", "w", []string{"localhost:5556"},
		"List of workers on which the manager will schedule tasks")
	managerCmd.Flags().StringP("scheduler", "s", "epvm", "Name of scheduler to use")
	managerCmd.Flags().StringP("dbtype", "d", "memory",
		"Type of datastore to use for events and tasks \"memory\" or \"persistent\"")
	managerCmd.Flags().StringP("datadir", "D", "./data", "Directory to store persistent data")
}
