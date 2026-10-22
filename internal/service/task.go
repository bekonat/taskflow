package service

import (
	"context"

	"github.com/bekonat/taskflow/internal/domain"
)

type TaskService struct {
	tasks TaskRepository
}

func NewTaskService(tasks TaskRepository) *TaskService {
	return &TaskService{tasks: tasks}
}

func (s *TaskService) Create(ctx context.Context, ownerID int64, task domain.Task) (*domain.Task, error) {
	task.OwnerID = ownerID
	if task.Status == "" {
		task.Status = domain.StatusTodo
	}
	if task.Priority == 0 {
		task.Priority = domain.PriorityMedium
	}
	if err := task.Validate(); err != nil {
		return nil, err
	}
	if err := s.tasks.Create(ctx, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *TaskService) Get(ctx context.Context, ownerID, id int64) (*domain.Task, error) {
	task, err := s.tasks.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if task.OwnerID != ownerID {
		return nil, domain.ErrNotFound
	}
	return task, nil
}

func (s *TaskService) List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, int, error) {
	if filter.Status != nil && !filter.Status.Valid() {
		return nil, 0, &domain.ValidationError{Field: "status", Message: "must be one of todo, in_progress, done"}
	}
	filter.Normalize()
	return s.tasks.List(ctx, filter)
}

func (s *TaskService) Update(ctx context.Context, ownerID, id int64, patch domain.TaskPatch) (*domain.Task, error) {
	task, err := s.Get(ctx, ownerID, id)
	if err != nil {
		return nil, err
	}
	patch.Apply(task)
	if err := task.Validate(); err != nil {
		return nil, err
	}
	if err := s.tasks.Update(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) Delete(ctx context.Context, ownerID, id int64) error {
	if _, err := s.Get(ctx, ownerID, id); err != nil {
		return err
	}
	return s.tasks.Delete(ctx, id)
}

func (s *TaskService) Stats(ctx context.Context, ownerID int64) (domain.TaskStats, error) {
	return s.tasks.Stats(ctx, ownerID)
}
