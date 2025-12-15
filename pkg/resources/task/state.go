package task

import (
	"slices"
	"time"
)

// type State represents 5 states of task
type State int

/*
Task can have 5 state:

	pending
	Scheduled
	running
	completed
	failed
*/
const (
	// the initial state, the starting point for every task
	Pending State = iota
	// a task moves to this state once the manager has scheduled it onto a worker
	Scheduled
	// a task moves to this state when a worker successfully starts the tasks
	Running
	// a task moves to this state when is completes its work in a normal way
	Completed
	// if a task fails it moves to this state
	Failed
)

var stateTransitionMap = map[State][]State {
	Pending: []State{Scheduled},
	Scheduled: []State{Scheduled, Running, Failed},
	Running: []State{Running, Completed, Failed},
	Completed: []State{},
	Failed: []State{},
}

func ValidateTransition(from, to State) bool {
	return slices.Contains(stateTransitionMap[from], to)
}

func StatePending(task Task) {
	setState(&task, Pending)
}

func StateScheduled(task Task) {
	setState(&task, Scheduled) 
}

func StateRunning(task Task) {
	setState(&task, Running)
}

func StateCompleted(task Task) {
	setState(&task, Completed)

	// enter logic
	task.FinishTime = time.Now().UTC()
}

func StateFailed(task Task) {
	switch task.State {
	case Pending:
	case Scheduled:
	}

	setState(&task, Failed)
}

func setState(task *Task, newValue State) {
	// exit logic
	switch task.State {
	case Pending:
	case Scheduled:
	case Running:
	case Completed:
	case Failed:
	}

	task.State = newValue
}
