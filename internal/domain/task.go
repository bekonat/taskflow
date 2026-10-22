package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

func (s Status) Valid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone:
		return true
	}
	return false
}

type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 2
	PriorityHigh   Priority = 3
)

func (p Priority) Valid() bool {
	return p >= PriorityLow && p <= PriorityHigh
}

type Task struct {
	ID          int64      `json:"id"`
	OwnerID     int64      `json:"owner_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	Priority    Priority   `json:"priority"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (t *Task) Validate() error {
	t.Title = strings.TrimSpace(t.Title)
	if t.Title == "" {
		return &ValidationError{Field: "title", Message: "is required"}
	}
	if utf8.RuneCountInString(t.Title) > 200 {
		return &ValidationError{Field: "title", Message: "must be at most 200 characters"}
	}
	if utf8.RuneCountInString(t.Description) > 5000 {
		return &ValidationError{Field: "description", Message: "must be at most 5000 characters"}
	}
	if !t.Status.Valid() {
		return &ValidationError{Field: "status", Message: "must be one of todo, in_progress, done"}
	}
	if !t.Priority.Valid() {
		return &ValidationError{Field: "priority", Message: "must be between 1 and 3"}
	}
	return nil
}

type TaskPatch struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Status      *Status    `json:"status"`
	Priority    *Priority  `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
}

func (p TaskPatch) Apply(t *Task) {
	if p.Title != nil {
		t.Title = *p.Title
	}
	if p.Description != nil {
		t.Description = *p.Description
	}
	if p.Status != nil {
		t.Status = *p.Status
	}
	if p.Priority != nil {
		t.Priority = *p.Priority
	}
	if p.DueDate != nil {
		t.DueDate = p.DueDate
	}
}

type TaskFilter struct {
	OwnerID int64
	Status  *Status
	Search  string
	Limit   int
	Offset  int
}

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

func (f *TaskFilter) Normalize() {
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	f.Search = strings.TrimSpace(f.Search)
}

type TaskStats struct {
	Total      int `json:"total"`
	Todo       int `json:"todo"`
	InProgress int `json:"in_progress"`
	Done       int `json:"done"`
	Overdue    int `json:"overdue"`
}
