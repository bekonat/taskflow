package service

import (
	"context"

	"github.com/bekonat/taskflow/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
}

type TaskRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id int64) (*domain.Task, error)
	List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, int, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id int64) error
	Stats(ctx context.Context, ownerID int64) (domain.TaskStats, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) (bool, error)
}

type TokenIssuer interface {
	Issue(userID int64) (string, error)
}
