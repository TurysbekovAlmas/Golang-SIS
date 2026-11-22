package task

import "errors"

var (
	ErrTaskNotFound       = errors.New("task not found")
	ErrInvalidInput       = errors.New("task input is invalid: text cannot be empty or whitespace-only")
	ErrInvalidStateChange = errors.New("invalid task state transition")
	ErrAlreadyCompleted   = errors.New("task is already completed")
	ErrQueueFull          = errors.New("task queue is full")
	ErrPoolStopped        = errors.New("worker pool is stopped")
	ErrPoolNotStarted     = errors.New("worker pool is not started")
	ErrInvalidPoolConfig  = errors.New("worker count and queue capacity must be greater than zero")
)

type TaskState string

const (
	StatePending    TaskState = "Pending"
	StateQueued     TaskState = "Queued"
	StateProcessing TaskState = "Processing"
	StateCompleted  TaskState = "Completed"
	StateFailed     TaskState = "Failed"
)

type TaskInput struct {
	Text string
}

type TaskResult struct {
	WordCount      int
	CharacterCount int
	RuneCount      int
	LineCount      int
}

type Task struct {
	ID     string
	Input  TaskInput
	State  TaskState
	Result *TaskResult
}

func NewTask(id string, input TaskInput) *Task {
	return &Task{
		ID:    id,
		Input: input,
		State: StatePending,
	}
}

func (t *Task) Clone() *Task {
	if t == nil {
		return nil
	}
	cp := *t
	if t.Result != nil {
		resCopy := *t.Result
		cp.Result = &resCopy
	}
	return &cp
}
