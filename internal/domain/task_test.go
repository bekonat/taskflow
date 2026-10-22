package domain

import (
	"errors"
	"strings"
	"testing"
)

func validTask() Task {
	return Task{Title: "Write tests", Status: StatusTodo, Priority: PriorityMedium}
}

func TestTaskValidate(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*Task)
		wantField string
	}{
		{"valid", func(*Task) {}, ""},
		{"empty title", func(t *Task) { t.Title = "   " }, "title"},
		{"long title", func(t *Task) { t.Title = strings.Repeat("a", 201) }, "title"},
		{"long description", func(t *Task) { t.Description = strings.Repeat("a", 5001) }, "description"},
		{"bad status", func(t *Task) { t.Status = "archived" }, "status"},
		{"bad priority", func(t *Task) { t.Priority = 7 }, "priority"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := validTask()
			tt.mutate(&task)
			err := task.Validate()
			if tt.wantField == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var ve *ValidationError
			if !errors.As(err, &ve) || ve.Field != tt.wantField {
				t.Fatalf("err = %v, want validation error on %q", err, tt.wantField)
			}
		})
	}
}

func TestTaskValidateTrimsTitle(t *testing.T) {
	task := validTask()
	task.Title = "  hello  "
	if err := task.Validate(); err != nil {
		t.Fatal(err)
	}
	if task.Title != "hello" {
		t.Fatalf("Title = %q, want %q", task.Title, "hello")
	}
}

func TestTaskPatchApply(t *testing.T) {
	task := validTask()
	title := "new"
	status := StatusDone
	TaskPatch{Title: &title, Status: &status}.Apply(&task)

	if task.Title != "new" || task.Status != StatusDone || task.Priority != PriorityMedium {
		t.Fatalf("unexpected task after patch: %+v", task)
	}
}

func TestTaskFilterNormalize(t *testing.T) {
	tests := []struct {
		in   TaskFilter
		want TaskFilter
	}{
		{TaskFilter{}, TaskFilter{Limit: DefaultLimit}},
		{TaskFilter{Limit: 500, Offset: -3}, TaskFilter{Limit: MaxLimit}},
		{TaskFilter{Limit: 10, Offset: 30, Search: " go "}, TaskFilter{Limit: 10, Offset: 30, Search: "go"}},
	}
	for _, tt := range tests {
		got := tt.in
		got.Normalize()
		if got.Limit != tt.want.Limit || got.Offset != tt.want.Offset || got.Search != tt.want.Search {
			t.Errorf("Normalize(%+v) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestCredentialsValidate(t *testing.T) {
	tests := []struct {
		name  string
		creds Credentials
		ok    bool
	}{
		{"valid", Credentials{Email: "user@example.com", Password: "secret123"}, true},
		{"bad email", Credentials{Email: "nope", Password: "secret123"}, false},
		{"short password", Credentials{Email: "user@example.com", Password: "123"}, false},
		{"long password", Credentials{Email: "user@example.com", Password: strings.Repeat("x", 73)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.creds.Validate(); (err == nil) != tt.ok {
				t.Fatalf("Validate() = %v, ok = %v", err, tt.ok)
			}
		})
	}
}

func TestCredentialsNormalize(t *testing.T) {
	c := Credentials{Email: "  User@Example.COM "}
	c.Normalize()
	if c.Email != "user@example.com" {
		t.Fatalf("Email = %q", c.Email)
	}
}
