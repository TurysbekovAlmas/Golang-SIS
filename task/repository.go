package task

type Repository interface {
	Save(task *Task) error
	GetByID(id string) (*Task, error)
}

type InMemoryRepository struct {
	tasks map[string]*Task
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		tasks: make(map[string]*Task),
	}
}

func (r *InMemoryRepository) Save(t *Task) error {
	r.tasks[t.ID] = t
	return nil
}

func (r *InMemoryRepository) GetByID(id string) (*Task, error) {
	t, ok := r.tasks[id]
	if !ok {
		return nil, ErrTaskNotFound
	}
	return t, nil
}
