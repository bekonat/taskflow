package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bekonat/taskflow/internal/domain"
)

const taskColumns = `id, owner_id, title, description, status, priority, due_date, created_at, updated_at`

type TaskRepository struct {
	pool *pgxpool.Pool
}

func NewTaskRepository(pool *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

func (r *TaskRepository) Create(ctx context.Context, t *domain.Task) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO tasks (owner_id, title, description, status, priority, due_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`,
		t.OwnerID, t.Title, t.Description, t.Status, t.Priority, t.DueDate,
	).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*domain.Task, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+taskColumns+` FROM tasks WHERE id = $1`, id)
	t, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return t, err
}

func (r *TaskRepository) List(ctx context.Context, f domain.TaskFilter) ([]domain.Task, int, error) {
	where, args := buildTaskWhere(f)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM tasks `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(`SELECT %s FROM tasks %s ORDER BY priority DESC, id DESC LIMIT $%d OFFSET $%d`,
		taskColumns, where, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	tasks := make([]domain.Task, 0, f.Limit)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, *t)
	}
	return tasks, total, rows.Err()
}

func (r *TaskRepository) Update(ctx context.Context, t *domain.Task) error {
	err := r.pool.QueryRow(ctx, `
		UPDATE tasks
		SET title = $2, description = $3, status = $4, priority = $5, due_date = $6, updated_at = now()
		WHERE id = $1
		RETURNING updated_at`,
		t.ID, t.Title, t.Description, t.Status, t.Priority, t.DueDate,
	).Scan(&t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	return err
}

func (r *TaskRepository) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *TaskRepository) Stats(ctx context.Context, ownerID int64) (domain.TaskStats, error) {
	var s domain.TaskStats
	err := r.pool.QueryRow(ctx, `
		SELECT
			count(*),
			count(*) FILTER (WHERE status = 'todo'),
			count(*) FILTER (WHERE status = 'in_progress'),
			count(*) FILTER (WHERE status = 'done'),
			count(*) FILTER (WHERE status <> 'done' AND due_date < now())
		FROM tasks
		WHERE owner_id = $1`, ownerID,
	).Scan(&s.Total, &s.Todo, &s.InProgress, &s.Done, &s.Overdue)
	return s, err
}

func buildTaskWhere(f domain.TaskFilter) (string, []any) {
	conds := []string{"owner_id = $1"}
	args := []any{f.OwnerID}
	if f.Status != nil {
		args = append(args, *f.Status)
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	if f.Search != "" {
		args = append(args, "%"+escapeLike(f.Search)+"%")
		conds = append(conds, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", len(args), len(args)))
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func scanTask(row pgx.Row) (*domain.Task, error) {
	var t domain.Task
	err := row.Scan(&t.ID, &t.OwnerID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.DueDate, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
