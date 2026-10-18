package task

import "errors"

var (
	ErrTaskNotFound       = errors.New("task not found")
	ErrInvalidInput       = errors.New("task input is invalid: text cannot be empty or whitespace-only")
	ErrInvalidStateChange = errors.New("invalid task state transition")
	ErrAlreadyCompleted   = errors.New("task is already completed")
)

type TaskState string

const (
	StatePending    TaskState = "Pending"
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
