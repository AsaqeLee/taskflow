package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
)

type MemoryTaskRepository struct {
	mu     sync.RWMutex
	tasks  map[string]domaintask.Task
	nextID int
}

func NewMemoryTaskRepository() *MemoryTaskRepository {
	return &MemoryTaskRepository{
		tasks:  make(map[string]domaintask.Task),
		nextID: 1,
	}
}

func (r *MemoryTaskRepository) Create(ctx context.Context, task domaintask.Task) (domaintask.Task, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domaintask.Task{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if task.ID() == "" {
		task = task.AssignID(fmt.Sprintf("task_%03d", r.nextID))
		r.nextID++
	}
	r.tasks[task.ID()] = task
	return task, nil
}

func (r *MemoryTaskRepository) GetByID(ctx context.Context, id string) (domaintask.Task, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domaintask.Task{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	if !ok {
		return domaintask.Task{}, ErrTaskNotFound
	}
	if task.IsDeleted() {
		return domaintask.Task{}, ErrTaskNotFound
	}
	return task, nil
}

func (r *MemoryTaskRepository) GetByIDIncludingDeleted(ctx context.Context, id string) (domaintask.Task, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domaintask.Task{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	if !ok {
		return domaintask.Task{}, ErrTaskNotFound
	}
	return task, nil
}

func (r *MemoryTaskRepository) List(ctx context.Context) ([]domaintask.Task, error) {
	if err := errIfContextDone(ctx); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domaintask.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		if task.IsDeleted() {
			continue
		}
		result = append(result, task)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt().Before(result[j].CreatedAt())
	})
	return result, nil
}

func (r *MemoryTaskRepository) ListVisibleToUser(ctx context.Context, userID string) ([]domaintask.Task, error) {
	if err := errIfContextDone(ctx); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domaintask.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		if task.IsDeleted() {
			continue
		}
		if task.CreatorID() == userID || task.AssigneeID() == userID {
			result = append(result, task)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt().Before(result[j].CreatedAt())
	})
	return result, nil
}

func (r *MemoryTaskRepository) Update(ctx context.Context, task domaintask.Task) (domaintask.Task, error) {
	if err := errIfContextDone(ctx); err != nil {
		return domaintask.Task{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.tasks[task.ID()]; !ok {
		return domaintask.Task{}, ErrTaskNotFound
	}

	r.tasks[task.ID()] = task
	return task, nil
}

func timeNowUTC() time.Time {
	return time.Now().UTC()
}
