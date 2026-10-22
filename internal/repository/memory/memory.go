package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bekonat/taskflow/internal/domain"
)

type UserRepository struct {
	mu     sync.RWMutex
	nextID int64
	byID   map[int64]domain.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{byID: make(map[int64]domain.User)}
}

func (r *UserRepository) Create(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, u := range r.byID {
		if u.Email == user.Email {
			return domain.ErrAlreadyExists
		}
	}
	r.nextID++
	user.ID = r.nextID
	user.CreatedAt = time.Now().UTC()
	r.byID[user.ID] = *user
	return nil
}

func (r *UserRepository) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, u := range r.byID {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *UserRepository) GetByID(_ context.Context, id int64) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	u, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &u, nil
}

type TaskRepository struct {
	mu     sync.RWMutex
	nextID int64
	byID   map[int64]domain.Task
	now    func() time.Time
}

func NewTaskRepository() *TaskRepository {
	return &TaskRepository{
		byID: make(map[int64]domain.Task),
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (r *TaskRepository) Create(_ context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	task.ID = r.nextID
	task.CreatedAt = r.now()
	task.UpdatedAt = task.CreatedAt
	r.byID[task.ID] = *task
	return nil
}

func (r *TaskRepository) GetByID(_ context.Context, id int64) (*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &t, nil
}

func (r *TaskRepository) List(_ context.Context, f domain.TaskFilter) ([]domain.Task, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	search := strings.ToLower(f.Search)
	var matched []domain.Task
	for _, t := range r.byID {
		if t.OwnerID != f.OwnerID {
			continue
		}
		if f.Status != nil && t.Status != *f.Status {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(t.Title), search) &&
			!strings.Contains(strings.ToLower(t.Description), search) {
			continue
		}
		matched = append(matched, t)
	}
	sort.Slice(matched, func(i, j int) bool {
		if matched[i].Priority != matched[j].Priority {
			return matched[i].Priority > matched[j].Priority
		}
		return matched[i].ID > matched[j].ID
	})

	total := len(matched)
	start := min(f.Offset, total)
	end := min(start+f.Limit, total)
	return matched[start:end], total, nil
}

func (r *TaskRepository) Update(_ context.Context, task *domain.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byID[task.ID]; !ok {
		return domain.ErrNotFound
	}
	task.UpdatedAt = r.now()
	r.byID[task.ID] = *task
	return nil
}

func (r *TaskRepository) Delete(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.byID, id)
	return nil
}

func (r *TaskRepository) Stats(_ context.Context, ownerID int64) (domain.TaskStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var s domain.TaskStats
	now := r.now()
	for _, t := range r.byID {
		if t.OwnerID != ownerID {
			continue
		}
		s.Total++
		switch t.Status {
		case domain.StatusTodo:
			s.Todo++
		case domain.StatusInProgress:
			s.InProgress++
		case domain.StatusDone:
			s.Done++
		}
		if t.Status != domain.StatusDone && t.DueDate != nil && t.DueDate.Before(now) {
			s.Overdue++
		}
	}
	return s, nil
}
