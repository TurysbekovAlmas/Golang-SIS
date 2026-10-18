package task

import (
	"fmt"
	"sync/atomic"
)

type Service struct {
	repo      Repository
	processor TaskProcessor
	idCounter uint64
}

func NewService(repo Repository, processor TaskProcessor) *Service {
	return &Service{
		repo:      repo,
		processor: processor,
	}
}

func (s *Service) generateID() string {
	id := atomic.AddUint64(&s.idCounter, 1)
	return fmt.Sprintf("TASK-%04d", id)
}

func (s *Service) CreateTask(input TaskInput) (*Task, error) {
	if err := s.processor.Validate(input); err != nil {
		return nil, err
	}

	id := s.generateID()
	t := NewTask(id, input)

	if err := s.repo.Save(t); err != nil {
		return nil, err
	}

	return t, nil
}

func (s *Service) GetTask(id string) (*Task, error) {
	return s.repo.GetByID(id)
}

func (s *Service) ProcessTask(id string) (*Task, error) {
	t, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if t.State == StateCompleted {
		return nil, ErrAlreadyCompleted
	}
	if t.State != StatePending {
		return nil, ErrInvalidStateChange
	}

	t.State = StateProcessing
	_ = s.repo.Save(t)

	res, err := s.processor.Process(t.Input)
	if err != nil {
		t.State = StateFailed
		_ = s.repo.Save(t)
		return nil, err
	}

	t.Result = &res
	t.State = StateCompleted
	_ = s.repo.Save(t)

	return t, nil
}
