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
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)
