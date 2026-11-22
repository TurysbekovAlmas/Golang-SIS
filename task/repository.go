package task

import "sync"

type Repository interface {
	Save(task *Task) error
	GetByID(id string) (*Task, error)
}

type InMemoryRepository struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		tasks: make(map[string]*Task),
	}
}

func (r *InMemoryRepository) Save(t *Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tasks[t.ID] = t.Clone()
	return nil
}

func (r *InMemoryRepository) GetByID(id string) (*Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return t.Clone(), nil
}
