package task

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func setupService() *Service {
	repo := NewInMemoryRepository()
	processor := NewTextAnalyzer()
	return NewService(repo, processor)
}

func TestTaskValidation_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		input   TaskInput
		wantErr error
	}{
		{
			name:    "Valid text input",
			input:   TaskInput{Text: "Hello Go World"},
			wantErr: nil,
		},
		{
			name:    "Empty text input",
			input:   TaskInput{Text: ""},
			wantErr: ErrInvalidInput,
		},
		{
			name:    "Whitespace-only input",
			input:   TaskInput{Text: "   \t\n  "},
			wantErr: ErrInvalidInput,
		},
	}

	svc := setupService()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateTask(tt.input)
			if err != tt.wantErr {
				t.Errorf("CreateTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateAndGetTask(t *testing.T) {
	svc := setupService()

	input := TaskInput{Text: "Testing Go applications"}
	createdTask, err := svc.CreateTask(input)
	if err != nil {
		t.Fatalf("unexpected error creating task: %v", err)
	}

	if createdTask.ID == "" {
		t.Errorf("expected non-empty task ID")
	}

	if createdTask.State != StatePending {
		t.Errorf("expected state %s, got %s", StatePending, createdTask.State)
	}

	fetchedTask, err := svc.GetTask(createdTask.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching task: %v", err)
	}

	if fetchedTask.ID != createdTask.ID {
		t.Errorf("expected ID %s, got %s", createdTask.ID, fetchedTask.ID)
	}
}

func TestGetNonExistentTask(t *testing.T) {
	svc := setupService()

	_, err := svc.GetTask("NON-EXISTENT-ID")
	if err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestProcessTask_Success(t *testing.T) {
	svc := setupService()

	input := TaskInput{Text: "Hello\nWorld"}
	tTask, err := svc.CreateTask(input)
	if err != nil {
		t.Fatalf("failed to create task: %v", err)
	}

	processedTask, err := svc.ProcessTask(tTask.ID)
	if err != nil {
		t.Fatalf("failed to process task: %v", err)
	}

	if processedTask.State != StateCompleted {
		t.Errorf("expected state %s, got %s", StateCompleted, processedTask.State)
	}

	if processedTask.Result == nil {
		t.Fatalf("expected non-nil result")
	}

	res := processedTask.Result
	if res.WordCount != 2 {
		t.Errorf("expected 2 words, got %d", res.WordCount)
	}
	if res.LineCount != 2 {
		t.Errorf("expected 2 lines, got %d", res.LineCount)
	}
}

func TestProcessAlreadyCompletedTask(t *testing.T) {
	svc := setupService()

	tTask, _ := svc.CreateTask(TaskInput{Text: "Sample text"})
	_, _ = svc.ProcessTask(tTask.ID)
	_, err := svc.ProcessTask(tTask.ID)
	if err != ErrAlreadyCompleted {
		t.Errorf("expected ErrAlreadyCompleted, got %v", err)
	}
}

func TestMultipleIndependentTasks(t *testing.T) {
	svc := setupService()

	task1, _ := svc.CreateTask(TaskInput{Text: "One"})
	task2, _ := svc.CreateTask(TaskInput{Text: "Two three"})

	if task1.ID == task2.ID {
		t.Errorf("expected unique IDs, got identical ID: %s", task1.ID)
	}

	_, _ = svc.ProcessTask(task1.ID)

	t1, _ := svc.GetTask(task1.ID)
	t2, _ := svc.GetTask(task2.ID)

	if t1.State != StateCompleted {
		t.Errorf("expected task1 state %s, got %s", StateCompleted, t1.State)
	}
	if t2.State != StatePending {
		t.Errorf("expected task2 state %s, got %s", StatePending, t2.State)
	}
	if t2.Result != nil {
		t.Errorf("expected task2 result to be nil before processing")
	}
}

type controlledProcessor struct {
	delegate *TextAnalyzer
	onStart  func(input TaskInput)
	release  <-chan struct{}
}

func (p *controlledProcessor) Validate(input TaskInput) error {
	return p.delegate.Validate(input)
}

func (p *controlledProcessor) Process(input TaskInput) (TaskResult, error) {
	if p.onStart != nil {
		p.onStart(input)
	}
	if p.release != nil {
		<-p.release
	}
	return p.delegate.Process(input)
}

func TestConcurrentWorkerPool_MultipleTasksAndResults(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithConfig(repo, NewTextAnalyzer(), 3, 20)

	if err := svc.StartWorkers(); err != nil {
		t.Fatalf("StartWorkers failed: %v", err)
	}

	const totalTasks = 15
	ids := make([]string, totalTasks)
	for i := 0; i < totalTasks; i++ {
		taskObj, err := svc.CreateTask(TaskInput{Text: fmt.Sprintf("word%d second third", i)})
		if err != nil {
			t.Fatalf("CreateTask failed: %v", err)
		}
		ids[i] = taskObj.ID
		if err := svc.SubmitTask(taskObj.ID); err != nil {
			t.Fatalf("SubmitTask failed: %v", err)
		}
	}

	svc.Shutdown()

	for _, id := range ids {
		got, err := svc.GetTask(id)
		if err != nil {
			t.Fatalf("GetTask(%s) failed: %v", id, err)
		}
		if got.State != StateCompleted {
			t.Errorf("task %s state = %s, want %s", id, got.State, StateCompleted)
		}
		if got.Result == nil || got.Result.WordCount != 3 {
			t.Errorf("task %s unexpected result: %+v", id, got.Result)
		}
	}
}

func TestConcurrentOverlappingExecution(t *testing.T) {
	repo := NewInMemoryRepository()
	startedCh := make(chan struct{}, 2)
	releaseCh := make(chan struct{})

	proc := &controlledProcessor{
		delegate: NewTextAnalyzer(),
		onStart: func(input TaskInput) {
			startedCh <- struct{}{}
		},
		release: releaseCh,
	}

	svc := NewServiceWithConfig(repo, proc, 2, 4)
	if err := svc.StartWorkers(); err != nil {
		t.Fatalf("StartWorkers failed: %v", err)
	}
	defer svc.Shutdown()

	t1, _ := svc.CreateTask(TaskInput{Text: "First task"})
	t2, _ := svc.CreateTask(TaskInput{Text: "Second task"})

	if err := svc.SubmitTask(t1.ID); err != nil {
		t.Fatalf("SubmitTask t1 failed: %v", err)
	}
	if err := svc.SubmitTask(t2.ID); err != nil {
		t.Fatalf("SubmitTask t2 failed: %v", err)
	}

	for i := 0; i < 2; i++ {
		select {
		case <-startedCh:
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for concurrent workers to overlap; only %d started", i)
		}
	}

	got1, err := svc.GetTask(t1.ID)
	if err != nil || got1.State != StateProcessing {
		t.Errorf("expected t1 to be Processing while blocked, got %v (err=%v)", got1, err)
	}

	close(releaseCh)
}

func TestBoundedQueueFullRejection(t *testing.T) {
	repo := NewInMemoryRepository()
	startedCh := make(chan struct{}, 1)
	releaseCh := make(chan struct{})

	proc := &controlledProcessor{
		delegate: NewTextAnalyzer(),
		onStart: func(input TaskInput) {
			select {
			case startedCh <- struct{}{}:
			default:
			}
		},
		release: releaseCh,
	}

	svc := NewServiceWithConfig(repo, proc, 1, 2)
	if err := svc.StartWorkers(); err != nil {
		t.Fatalf("StartWorkers failed: %v", err)
	}
	defer func() {
		close(releaseCh)
		svc.Shutdown()
	}()

	t1, _ := svc.CreateTask(TaskInput{Text: "Task 1"})
	t2, _ := svc.CreateTask(TaskInput{Text: "Task 2"})
	t3, _ := svc.CreateTask(TaskInput{Text: "Task 3"})
	t4, _ := svc.CreateTask(TaskInput{Text: "Task 4"})

	if err := svc.SubmitTask(t1.ID); err != nil {
		t.Fatalf("t1 submit failed: %v", err)
	}
	<-startedCh

	if err := svc.SubmitTask(t2.ID); err != nil {
		t.Fatalf("t2 submit failed: %v", err)
	}
	if err := svc.SubmitTask(t3.ID); err != nil {
		t.Fatalf("t3 submit failed: %v", err)
	}

	err := svc.SubmitTask(t4.ID)
	if err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}

	got4, _ := svc.GetTask(t4.ID)
	if got4.State != StatePending {
		t.Errorf("expected rejected task to remain Pending, got %s", got4.State)
	}
}

func TestDuplicateAndInvalidSubmissions(t *testing.T) {
	repo := NewInMemoryRepository()
	releaseCh := make(chan struct{})
	proc := &controlledProcessor{
		delegate: NewTextAnalyzer(),
		release:  releaseCh,
	}

	svc := NewServiceWithConfig(repo, proc, 2, 5)
	_ = svc.StartWorkers()

	if err := svc.SubmitTask("UNKNOWN-ID"); err != ErrTaskNotFound {
		t.Errorf("expected ErrTaskNotFound, got %v", err)
	}

	taskObj, _ := svc.CreateTask(TaskInput{Text: "Duplicate test"})
	var successCount atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := svc.SubmitTask(taskObj.ID); err == nil {
				successCount.Add(1)
			}
		}()
	}
	wg.Wait()

	if successCount.Load() != 1 {
		t.Errorf("expected exactly 1 successful submission, got %d", successCount.Load())
	}

	close(releaseCh)
	svc.Shutdown()

	svc2 := setupService()
	_ = svc2.StartWorkers()
	defer svc2.Shutdown()

	tDone, _ := svc2.CreateTask(TaskInput{Text: "Already done"})
	_, _ = svc2.ProcessTask(tDone.ID)
	if err := svc2.SubmitTask(tDone.ID); err != ErrAlreadyCompleted {
		t.Errorf("expected ErrAlreadyCompleted, got %v", err)
	}
}

func TestShutdownRejectsNewAndFinishesAccepted(t *testing.T) {
	repo := NewInMemoryRepository()
	svc := NewServiceWithConfig(repo, NewTextAnalyzer(), 2, 10)
	_ = svc.StartWorkers()

	var ids []string
	for i := 0; i < 6; i++ {
		tObj, _ := svc.CreateTask(TaskInput{Text: fmt.Sprintf("Task number %d", i)})
		if err := svc.SubmitTask(tObj.ID); err != nil {
			t.Fatalf("SubmitTask failed: %v", err)
		}
		ids = append(ids, tObj.ID)
	}

	svc.Shutdown()
	svc.Shutdown()

	for _, id := range ids {
		got, err := svc.GetTask(id)
		if err != nil || got.State != StateCompleted {
			t.Errorf("task %s not completed after shutdown: state=%v err=%v", id, got.State, err)
		}
	}

	lateTask, _ := svc.CreateTask(TaskInput{Text: "Late task"})
	if err := svc.SubmitTask(lateTask.ID); err != ErrPoolStopped {
		t.Errorf("expected ErrPoolStopped after shutdown, got %v", err)
	}
}
