package postgres

import (
	"slices"
	"testing"

	"github.com/bekonat/taskflow/internal/domain"
)

func TestBuildTaskWhere(t *testing.T) {
	done := domain.StatusDone
	tests := []struct {
		name      string
		filter    domain.TaskFilter
		wantWhere string
		wantArgs  []any
	}{
		{
			name:      "owner only",
			filter:    domain.TaskFilter{OwnerID: 7},
			wantWhere: "WHERE owner_id = $1",
			wantArgs:  []any{int64(7)},
		},
		{
			name:      "status and search",
			filter:    domain.TaskFilter{OwnerID: 7, Status: &done, Search: "go"},
			wantWhere: "WHERE owner_id = $1 AND status = $2 AND (title ILIKE $3 OR description ILIKE $3)",
			wantArgs:  []any{int64(7), done, "%go%"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			where, args := buildTaskWhere(tt.filter)
			if where != tt.wantWhere {
				t.Fatalf("where = %q, want %q", where, tt.wantWhere)
			}
			if !slices.Equal(args, tt.wantArgs) {
				t.Fatalf("args = %v, want %v", args, tt.wantArgs)
			}
		})
	}
}

func TestEscapeLike(t *testing.T) {
	if got := escapeLike(`50%_off\`); got != `50\%\_off\\` {
		t.Fatalf("escapeLike = %q", got)
	}
}
