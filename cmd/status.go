package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"orchestrator/pkg/resources/task"
	"os"
	"text/tabwriter"
	"time"

	"github.com/docker/go-units"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Status command to list tasks",
	Long:  `The status command allows a user to get the status of tasks from the manager`,
	Run: func(cmd *cobra.Command, args []string) {
		manager, _ := cmd.Flags().GetString("manager")

		url := fmt.Sprintf("http://%s/tasks", manager)
		resp, err := http.Get(url)
		if err != nil {
			log.Fatalf("Error connecting to manager %s: %v\n", manager, err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var tasks []*task.Task
		if err = json.Unmarshal(body, &tasks); err != nil {
			log.Fatal(err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 5, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "ID\tNAME\tCREATED\tSTATE\tCONTAINERNAME\tIMAGE\t")

		for _, task := range tasks {
			var start string
			if task.StartTime.IsZero() {
				start = "N/A"
			} else {
				start = fmt.Sprintf("%s ago", units.HumanDuration(time.Now().UTC().Sub(task.StartTime)))
			}

			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t\n", task.UUID, task.Name, start, task.State, task.Name, task.Image)
		}

		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringP("manager", "m", "localhost:5555", "Manager to talk to")
}
