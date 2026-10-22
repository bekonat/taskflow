package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bekonat/taskflow/internal/domain"
)

const uniqueViolation = "23505"

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id, created_at`,
		user.Email, user.PasswordHash,
	).Scan(&user.ID, &user.CreatedAt)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return domain.ErrAlreadyExists
	}
	return err
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.getOne(ctx, `SELECT id, email, password_hash, created_at FROM users WHERE email = $1`, email)
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	return r.getOne(ctx, `SELECT id, email, password_hash, created_at FROM users WHERE id = $1`, id)
}

func (r *UserRepository) getOne(ctx context.Context, query string, arg any) (*domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, query, arg).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
