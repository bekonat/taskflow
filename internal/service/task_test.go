package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bekonat/taskflow/internal/domain"
	"github.com/bekonat/taskflow/internal/repository/memory"
)

func ptr[T any](v T) *T { return &v }

func TestCreateAppliesDefaults(t *testing.T) {
	svc := NewTaskService(memory.NewTaskRepository())
	task, err := svc.Create(context.Background(), 1, domain.Task{Title: "Learn Go"})
	if err != nil {
		t.Fatal(err)
	}
	if task.ID == 0 || task.OwnerID != 1 || task.Status != domain.StatusTodo || task.Priority != domain.PriorityMedium {
		t.Fatalf("unexpected task: %+v", task)
	}
}

func TestCreateValidation(t *testing.T) {
	svc := NewTaskService(memory.NewTaskRepository())
	_, err := svc.Create(context.Background(), 1, domain.Task{Title: ""})
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}

func TestOwnershipIsEnforced(t *testing.T) {
	svc := NewTaskService(memory.NewTaskRepository())
	ctx := context.Background()
	task, _ := svc.Create(ctx, 1, domain.Task{Title: "private"})

	if _, err := svc.Get(ctx, 2, task.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Get by other user: err = %v", err)
	}
	if _, err := svc.Update(ctx, 2, task.ID, domain.TaskPatch{Title: ptr("hacked")}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Update by other user: err = %v", err)
	}
	if err := svc.Delete(ctx, 2, task.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Delete by other user: err = %v", err)
	}
	if got, _ := svc.Get(ctx, 1, task.ID); got.Title != "private" {
		t.Fatalf("task was modified: %+v", got)
	}
}

func TestUpdate(t *testing.T) {
	svc := NewTaskService(memory.NewTaskRepository())
	ctx := context.Background()
	task, _ := svc.Create(ctx, 1, domain.Task{Title: "draft"})

	updated, err := svc.Update(ctx, 1, task.ID, domain.TaskPatch{
		Status:   ptr(domain.StatusDone),
		Priority: ptr(domain.PriorityHigh),
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != domain.StatusDone || updated.Priority != domain.PriorityHigh || updated.Title != "draft" {
		t.Fatalf("unexpected task: %+v", updated)
	}

	_, err = svc.Update(ctx, 1, task.ID, domain.TaskPatch{Status: ptr(domain.Status("bogus"))})
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}

func TestDelete(t *testing.T) {
	svc := NewTaskService(memory.NewTaskRepository())
	ctx := context.Background()
	task, _ := svc.Create(ctx, 1, domain.Task{Title: "temp"})

	if err := svc.Delete(ctx, 1, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, 1, task.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestListFilteringAndPagination(t *testing.T) {
	svc := NewTaskService(memory.NewTaskRepository())
	ctx := context.Background()
	for i, title := range []string{"buy milk", "write report", "review PR", "buy bread"} {
		_, err := svc.Create(ctx, 1, domain.Task{Title: title, Priority: domain.Priority(i%3 + 1)})
		if err != nil {
			t.Fatal(err)
		}
	}
	svc.Create(ctx, 2, domain.Task{Title: "buy car"})

	tasks, total, err := svc.List(ctx, domain.TaskFilter{OwnerID: 1, Search: "buy"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(tasks) != 2 {
		t.Fatalf("search: total = %d, len = %d", total, len(tasks))
	}

	tasks, total, _ = svc.List(ctx, domain.TaskFilter{OwnerID: 1, Limit: 3, Offset: 3})
	if total != 4 || len(tasks) != 1 {
		t.Fatalf("pagination: total = %d, len = %d", total, len(tasks))
	}

	_, _, err = svc.List(ctx, domain.TaskFilter{OwnerID: 1, Status: ptr(domain.Status("x"))})
	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}

func TestStats(t *testing.T) {
	svc := NewTaskService(memory.NewTaskRepository())
	ctx := context.Background()
	past := time.Now().Add(-time.Hour)

	svc.Create(ctx, 1, domain.Task{Title: "a"})
	svc.Create(ctx, 1, domain.Task{Title: "b", Status: domain.StatusInProgress, DueDate: &past})
	svc.Create(ctx, 1, domain.Task{Title: "c", Status: domain.StatusDone, DueDate: &past})

	stats, err := svc.Stats(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.TaskStats{Total: 3, Todo: 1, InProgress: 1, Done: 1, Overdue: 1}
	if stats != want {
		t.Fatalf("Stats() = %+v, want %+v", stats, want)
	}
}
