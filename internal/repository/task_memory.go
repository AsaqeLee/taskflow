package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/AsaqeLee/taskflow/internal/model"
)

type MemoryTaskRepository struct {
	mu     sync.RWMutex
	tasks  map[string]model.Task
	nextID int
}

func NewMemoryTaskRepository() *MemoryTaskRepository {
	return &MemoryTaskRepository{
		tasks:  make(map[string]model.Task),
		nextID: 1,
	}
}

func (r *MemoryTaskRepository) Create(ctx context.Context, task model.Task) (model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if task.ID == "" {
		task.ID = fmt.Sprintf("task_%03d", r.nextID)
		r.nextID++
	}

	r.tasks[task.ID] = task
	return task, nil
}

func (r *MemoryTaskRepository) GetByID(ctx context.Context, id string) (model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	if !ok {
		return model.Task{}, ErrTaskNotFound
	}

	return task, nil
}

func (r *MemoryTaskRepository) List(ctx context.Context) ([]model.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		result = append(result, task)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result, nil
}

func (r *MemoryTaskRepository) Update(ctx context.Context, task model.Task) (model.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasks[task.ID]; !ok {
		return model.Task{}, ErrTaskNotFound
	}

	r.tasks[task.ID] = task
	return task, nil
}

func (r *MemoryTaskRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasks[id]; !ok {
		return ErrTaskNotFound
	}

	delete(r.tasks, id)
	return nil
}
