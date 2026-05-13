/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"orchestrator/pkg/resources/worker"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// workerCmd represents the worker command
var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Worker command to operate a orch worker node",
	Long:  `The worker runs tasks and responds to the manager's requests about task`,
	Run: func(cmd *cobra.Command, args []string) {
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		name, _ := cmd.Flags().GetString("name")
		dbtype, _ := cmd.Flags().GetString("dbtype")
		dataDir, _ := cmd.Flags().GetString("datadir")

		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("Failed to create data directory: %v", err)
		}

		router := gin.Default()
		w := worker.New(name, dbtype, dataDir)
		api := worker.Api{Host: host, Port: strconv.Itoa(port), Worker: w, Router: router}

		go w.RunTasks()
		go w.CollectStats()
		go w.UpdateTasks()

		log.Printf("Starting worker API on %s:%d", host, port)

		api.Register()

		if err := router.Run(fmt.Sprintf("%s:%d", host, port)); err != nil {
			log.Fatalf("Failed to start worker API: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)

	workerCmd.Flags().StringP("host", "H", "0.0.0.0", "Hostname or IP address")
	workerCmd.Flags().IntP("port", "p", 5556, "Port on which to listen")
	workerCmd.Flags().StringP("name", "n", fmt.Sprintf("worker-%s", uuid.New().String()), "Name of the worker")
	workerCmd.Flags().StringP("dbtype", "d", "memory", "Type of datastore for tasks (\"memory\" of \"persistent\")")
	workerCmd.Flags().StringP("datadir", "D", "./data", "Directory to store persistent data")
}
