package task

import (
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	DefaultWorkerCount   = 3
	DefaultQueueCapacity = 10
)

type Service struct {
	repo      Repository
	processor TaskProcessor
	idCounter uint64

	stateMu sync.Mutex

	poolMu        sync.Mutex
	queue         chan string
	workerCount   int
	queueCapacity int
	started       bool
	stopped       bool
	wg            sync.WaitGroup
}

func NewService(repo Repository, processor TaskProcessor) *Service {
	return NewServiceWithConfig(repo, processor, DefaultWorkerCount, DefaultQueueCapacity)
}

func NewServiceWithConfig(repo Repository, processor TaskProcessor, workers, queueCapacity int) *Service {
	if workers <= 0 {
		workers = DefaultWorkerCount
	}
	if queueCapacity <= 0 {
		queueCapacity = DefaultQueueCapacity
	}
	return &Service{
		repo:          repo,
		processor:     processor,
		workerCount:   workers,
		queueCapacity: queueCapacity,
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

func (s *Service) StartWorkers() error {
	s.poolMu.Lock()
	defer s.poolMu.Unlock()

	if s.stopped {
		return ErrPoolStopped
	}
	if s.started {
		return nil
	}

	s.queue = make(chan string, s.queueCapacity)
	s.started = true

	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.workerLoop()
	}

	return nil
}

func (s *Service) workerLoop() {
	defer s.wg.Done()
	for id := range s.queue {
		s.executeTask(id)
	}
}
func (s *Service) SubmitTask(id string) error {
	s.poolMu.Lock()
	if !s.started {
		s.poolMu.Unlock()
		return ErrPoolNotStarted
	}
	if s.stopped {
		s.poolMu.Unlock()
		return ErrPoolStopped
	}
	s.stateMu.Lock()
	t, err := s.repo.GetByID(id)
	if err != nil {
		s.stateMu.Unlock()
		s.poolMu.Unlock()
		return err
	}
	if t.State == StateCompleted {
		s.stateMu.Unlock()
		s.poolMu.Unlock()
		return ErrAlreadyCompleted
	}
	if t.State != StatePending {
		s.stateMu.Unlock()
		s.poolMu.Unlock()
		return ErrInvalidStateChange
	}

	t.State = StateQueued
	_ = s.repo.Save(t)
	s.stateMu.Unlock()

	select {
	case s.queue <- id:
		s.poolMu.Unlock()
		return nil
	default:
		s.stateMu.Lock()
		if curr, getErr := s.repo.GetByID(id); getErr == nil && curr.State == StateQueued {
			curr.State = StatePending
			_ = s.repo.Save(curr)
		}
		s.stateMu.Unlock()
		s.poolMu.Unlock()
		return ErrQueueFull
	}
}

func (s *Service) CreateAndSubmitTask(input TaskInput) (*Task, error) {
	t, err := s.CreateTask(input)
	if err != nil {
		return nil, err
	}
	if err := s.SubmitTask(t.ID); err != nil {
		return t, err
	}
	return s.GetTask(t.ID)
}

func (s *Service) Shutdown() {
	s.poolMu.Lock()
	if !s.started || s.stopped {
		s.poolMu.Unlock()
		return
	}
	s.stopped = true
	close(s.queue)
	s.poolMu.Unlock()

	s.wg.Wait()
}
func (s *Service) executeTask(id string) {
	s.stateMu.Lock()
	t, err := s.repo.GetByID(id)
	if err != nil {
		s.stateMu.Unlock()
		return
	}
	if t.State != StateQueued && t.State != StatePending {
		s.stateMu.Unlock()
		return
	}
	t.State = StateProcessing
	_ = s.repo.Save(t)
	input := t.Input
	s.stateMu.Unlock()

	res, procErr := s.processor.Process(input)

	s.stateMu.Lock()
	curr, err := s.repo.GetByID(id)
	if err != nil {
		s.stateMu.Unlock()
		return
	}
	if procErr != nil {
		curr.State = StateFailed
		_ = s.repo.Save(curr)
		s.stateMu.Unlock()
		return
	}

	curr.Result = &res
	curr.State = StateCompleted
	_ = s.repo.Save(curr)
	s.stateMu.Unlock()
}

func (s *Service) ProcessTask(id string) (*Task, error) {
	s.stateMu.Lock()
	t, err := s.repo.GetByID(id)
	if err != nil {
		s.stateMu.Unlock()
		return nil, err
	}

	if t.State == StateCompleted {
		s.stateMu.Unlock()
		return nil, ErrAlreadyCompleted
	}
	if t.State != StatePending {
		s.stateMu.Unlock()
		return nil, ErrInvalidStateChange
	}

	t.State = StateProcessing
	_ = s.repo.Save(t)
	input := t.Input
	s.stateMu.Unlock()

	res, err := s.processor.Process(input)

	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	curr, getErr := s.repo.GetByID(id)
	if getErr != nil {
		return nil, getErr
	}
	if err != nil {
		curr.State = StateFailed
		_ = s.repo.Save(curr)
		return nil, err
	}

	curr.Result = &res
	curr.State = StateCompleted
	_ = s.repo.Save(curr)

	return curr, nil
}
