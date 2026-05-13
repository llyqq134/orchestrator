package cmd

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop a running task",
	Long:  `The stop command stops a running task`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			log.Fatal("Task UUID is required")
		}

		manager, _ := cmd.Flags().GetString("manager")

		url := fmt.Sprintf("http://%s/tasks/%s", manager, args[0])
		client := &http.Client{}

		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			log.Fatalf("Error creating request %v: %v\n", url, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("Error connecting to %v: %v\n", url, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			log.Fatalf("Error sending request, status: %v\n", resp.StatusCode)
		}

		log.Printf("Task %v was stopped\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().StringP("manager", "m", "localhost:5555", "Manager to talk to")
}
