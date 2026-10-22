package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/bekonat/taskflow/internal/domain"
)

type UserService interface {
	Register(ctx context.Context, creds domain.Credentials) (*domain.User, error)
	Login(ctx context.Context, creds domain.Credentials) (string, error)
	Get(ctx context.Context, id int64) (*domain.User, error)
}

type TaskService interface {
	Create(ctx context.Context, ownerID int64, task domain.Task) (*domain.Task, error)
	Get(ctx context.Context, ownerID, id int64) (*domain.Task, error)
	List(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, int, error)
	Update(ctx context.Context, ownerID, id int64, patch domain.TaskPatch) (*domain.Task, error)
	Delete(ctx context.Context, ownerID, id int64) error
	Stats(ctx context.Context, ownerID int64) (domain.TaskStats, error)
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	users  UserService
	tasks  TaskService
	tokens TokenParser
	health HealthChecker
	log    *slog.Logger
}

func NewHandler(users UserService, tasks TaskService, tokens TokenParser, health HealthChecker, log *slog.Logger) *Handler {
	return &Handler{users: users, tasks: tasks, tokens: tokens, health: health, log: log}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.HandleFunc("GET /api/v1/me", h.requireAuth(h.me))

	mux.HandleFunc("GET /api/v1/tasks", h.requireAuth(h.listTasks))
	mux.HandleFunc("POST /api/v1/tasks", h.requireAuth(h.createTask))
	mux.HandleFunc("GET /api/v1/tasks/stats", h.requireAuth(h.taskStats))
	mux.HandleFunc("GET /api/v1/tasks/{id}", h.requireAuth(h.getTask))
	mux.HandleFunc("PATCH /api/v1/tasks/{id}", h.requireAuth(h.updateTask))
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", h.requireAuth(h.deleteTask))

	var handler http.Handler = mux
	handler = logging(h.log)(handler)
	handler = recoverer(h.log)(handler)
	handler = requestID(handler)
	return handler
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	if h.health != nil {
		if err := h.health.Ping(r.Context()); err != nil {
			writeError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var creds domain.Credentials
	if err := decodeJSON(w, r, &creds); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	user, err := h.users.Register(r.Context(), creds)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var creds domain.Credentials
	if err := decodeJSON(w, r, &creds); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	token, err := h.users.Login(r.Context(), creds)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"access_token": token, "token_type": "Bearer"})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	user, err := h.users.Get(r.Context(), userIDFrom(r.Context()))
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

type taskListResponse struct {
	Items  []domain.Task `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := domain.TaskFilter{
		OwnerID: userIDFrom(r.Context()),
		Search:  q.Get("q"),
	}
	if s := q.Get("status"); s != "" {
		status := domain.Status(s)
		filter.Status = &status
	}
	var err error
	if filter.Limit, err = intParam(q.Get("limit")); err != nil {
		writeError(w, http.StatusBadRequest, "limit must be an integer")
		return
	}
	if filter.Offset, err = intParam(q.Get("offset")); err != nil {
		writeError(w, http.StatusBadRequest, "offset must be an integer")
		return
	}

	tasks, total, err := h.tasks.List(r.Context(), filter)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	filter.Normalize()
	writeJSON(w, http.StatusOK, taskListResponse{Items: tasks, Total: total, Limit: filter.Limit, Offset: filter.Offset})
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var task domain.Task
	if err := decodeJSON(w, r, &task); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	created, err := h.tasks.Create(r.Context(), userIDFrom(r.Context()), task)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	task, err := h.tasks.Get(r.Context(), userIDFrom(r.Context()), id)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var patch domain.TaskPatch
	if err := decodeJSON(w, r, &patch); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	task, err := h.tasks.Update(r.Context(), userIDFrom(r.Context()), id, patch)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.tasks.Delete(r.Context(), userIDFrom(r.Context()), id); err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) taskStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.tasks.Stats(r.Context(), userIDFrom(r.Context()))
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

func intParam(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}
