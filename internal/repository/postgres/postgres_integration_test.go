//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bekonat/taskflow/internal/domain"
	"github.com/bekonat/taskflow/migrations"
)

func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	if err := Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE users, tasks RESTART IDENTITY CASCADE`); err != nil {
		t.Fatal(err)
	}
	return pool
}

func createUser(t *testing.T, repo *UserRepository, email string) *domain.User {
	t.Helper()
	u := &domain.User{Email: email, PasswordHash: "hash"}
	if err := repo.Create(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestMigrateIsIdempotent(t *testing.T) {
	pool := setupDB(t)
	if err := Migrate(context.Background(), pool, migrations.FS); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestUserRepository(t *testing.T) {
	pool := setupDB(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	u := createUser(t, repo, "dev@example.com")
	if u.ID == 0 || u.CreatedAt.IsZero() {
		t.Fatalf("unexpected user: %+v", u)
	}

	if err := repo.Create(ctx, &domain.User{Email: "dev@example.com", PasswordHash: "x"}); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("duplicate: err = %v", err)
	}

	got, err := repo.GetByEmail(ctx, "dev@example.com")
	if err != nil || got.ID != u.ID {
		t.Fatalf("GetByEmail = %+v, %v", got, err)
	}
	if _, err := repo.GetByID(ctx, 999); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("GetByID(missing): err = %v", err)
	}
}

func TestTaskRepository(t *testing.T) {
	pool := setupDB(t)
	owner := createUser(t, NewUserRepository(pool), "owner@example.com")
	repo := NewTaskRepository(pool)
	ctx := context.Background()

	past := time.Now().Add(-24 * time.Hour)
	tasks := []domain.Task{
		{OwnerID: owner.ID, Title: "Learn pgx", Status: domain.StatusTodo, Priority: domain.PriorityHigh},
		{OwnerID: owner.ID, Title: "Write 100% tests", Status: domain.StatusInProgress, Priority: domain.PriorityLow, DueDate: &past},
		{OwnerID: owner.ID, Title: "Deploy", Status: domain.StatusDone, Priority: domain.PriorityMedium},
	}
	for i := range tasks {
		if err := repo.Create(ctx, &tasks[i]); err != nil {
			t.Fatal(err)
		}
	}

	list, total, err := repo.List(ctx, domain.TaskFilter{OwnerID: owner.ID, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || list[0].Title != "Learn pgx" || list[2].Title != "Write 100% tests" {
		t.Fatalf("List order/total wrong: total=%d %+v", total, list)
	}

	list, total, _ = repo.List(ctx, domain.TaskFilter{OwnerID: owner.ID, Search: "100%", Limit: 10})
	if total != 1 || len(list) != 1 {
		t.Fatalf("search with %% escape: total=%d", total)
	}

	task := tasks[0]
	task.Status = domain.StatusDone
	if err := repo.Update(ctx, &task); err != nil {
		t.Fatal(err)
	}
	got, _ := repo.GetByID(ctx, task.ID)
	if got.Status != domain.StatusDone {
		t.Fatalf("status after update = %s", got.Status)
	}

	stats, err := repo.Stats(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.TaskStats{Total: 3, Todo: 0, InProgress: 1, Done: 2, Overdue: 1}
	if stats != want {
		t.Fatalf("Stats = %+v, want %+v", stats, want)
	}

	if err := repo.Delete(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, task.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("second delete: err = %v", err)
	}
}
