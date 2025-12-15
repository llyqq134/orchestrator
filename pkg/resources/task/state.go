package task

// type State represents 4 states of task
type State int

/*
Task can have 4 state:

	pending
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
