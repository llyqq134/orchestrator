package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"orchestrator/pkg/metrics"
	"orchestrator/pkg/utils"
)

type Node struct {
	Name            string
	IpAddr          string
	Api             string
	Cores           int
	Memory          int
	MemoryAllocated int
	Disk            int
	DiskAllocated   int
	Stats           metrics.Stats
	Role            string
	TaskCount       int
}

func New(name, api, role string) *Node {
	return &Node{
		Name: name,
		Api:  api,
		Role: role,
	}
}

func (n *Node) GetStats() (*metrics.Stats, error) {
	op := "[node.GetStats]: "

	url := fmt.Sprintf("http://%s/stats", n.Api)
	resp, err := utils.HTTPWithRetry(http.Get, url)
	if err != nil {
		msg := fmt.Sprintf("Unable to connect to %v. Permanent failure", n.Api)
		log.Println(op + msg)

		return nil, errors.New(msg)
	}

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Error retrieving stats from %v: %v", n.Api, err)
		log.Println(op + msg)

		return nil, errors.New(msg)
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var stats metrics.Stats
	if err = json.Unmarshal(body, &stats); err != nil {
		msg := fmt.Sprintf("Error decoding message while getting stats for node %v", n.Name)
		log.Println(op + msg)

		return nil, err
	}

	return &n.Stats, nil
}
