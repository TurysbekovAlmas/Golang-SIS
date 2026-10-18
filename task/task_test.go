package task

import (
	"testing"
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

// 3. Обработка запроса несуществующей задачи (Requirement 4)
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
