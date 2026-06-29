package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AsaqeLee/taskflow/internal/domain/ports"
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
	result, err := r.Search(ctx, ports.TaskListQuery{})
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

func (r *MemoryTaskRepository) ListVisibleToUser(ctx context.Context, userID string) ([]domaintask.Task, error) {
	result, err := r.SearchVisibleToUser(ctx, userID, ports.TaskListQuery{})
	if err != nil {
		return nil, err
	}
	return result.Tasks, nil
}

func (r *MemoryTaskRepository) Search(ctx context.Context, query ports.TaskListQuery) (ports.TaskListResult, error) {
	return r.search(ctx, "", query)
}

func (r *MemoryTaskRepository) SearchVisibleToUser(ctx context.Context, userID string, query ports.TaskListQuery) (ports.TaskListResult, error) {
	return r.search(ctx, userID, query)
}

func (r *MemoryTaskRepository) search(ctx context.Context, userID string, query ports.TaskListQuery) (ports.TaskListResult, error) {
	if err := errIfContextDone(ctx); err != nil {
		return ports.TaskListResult{}, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	searchText := strings.ToLower(strings.TrimSpace(query.Query))
	status := strings.TrimSpace(query.Status)

	result := make([]domaintask.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		if task.IsDeleted() {
			continue
		}
		if userID != "" && task.CreatorID() != userID && task.AssigneeID() != userID {
			continue
		}
		if status != "" && task.Status().String() != status {
			continue
		}
		if searchText != "" {
			title := strings.ToLower(task.Title())
			description := strings.ToLower(task.Description())
			if !strings.Contains(title, searchText) && !strings.Contains(description, searchText) {
				continue
			}
		}
		result = append(result, task)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt().Before(result[j].CreatedAt())
	})

	total := len(result)
	start := query.Offset
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := total
	if query.Limit > 0 && start+query.Limit < end {
		end = start + query.Limit
	}

	return ports.TaskListResult{
		Tasks: result[start:end],
		Total: total,
	}, nil
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
