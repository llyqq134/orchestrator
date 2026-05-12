package scheduler

import (
	"orchestrator/pkg/resources/node"
	roundrobin "orchestrator/pkg/resources/scheduler/roundRobin"
	"orchestrator/pkg/resources/task"
)

const (
	RoundRobinScheduler = "roundRobin"
)

type Scheduler interface {
	SelectCandidateNodes(t task.Task, nodes []*node.Node) []*node.Node
	Score(t task.Task, nodes []*node.Node) map[string]float64
	Pick(scores map[string]float64, nodes []*node.Node) *node.Node
}

func New(kind string) Scheduler {
	switch kind {
	case RoundRobinScheduler:
		return &roundrobin.RoundRobin{
			Name: RoundRobinScheduler,
		}
	default:
		return &roundrobin.RoundRobin{
			Name: RoundRobinScheduler,
		}
	}
}
