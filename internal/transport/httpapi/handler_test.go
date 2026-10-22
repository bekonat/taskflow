package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/bekonat/taskflow/internal/auth"
	"github.com/bekonat/taskflow/internal/domain"
	"github.com/bekonat/taskflow/internal/repository/memory"
	"github.com/bekonat/taskflow/internal/service"
)

type testServer struct {
	t   *testing.T
	srv *httptest.Server
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	jwt := auth.NewJWTManager("test-secret", time.Hour)
	users := service.NewUserService(memory.NewUserRepository(), &auth.BcryptHasher{Cost: bcrypt.MinCost}, jwt)
	tasks := service.NewTaskService(memory.NewTaskRepository())
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewHandler(users, tasks, jwt, nil, log)

	srv := httptest.NewServer(h.Routes())
	t.Cleanup(srv.Close)
	return &testServer{t: t, srv: srv}
}

func (s *testServer) do(method, path, token string, body any) (*http.Response, map[string]any) {
	s.t.Helper()
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			s.t.Fatal(err)
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, s.srv.URL+path, buf)
	if err != nil {
		s.t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.srv.Client().Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer resp.Body.Close()

	var out map[string]any
	if resp.StatusCode != http.StatusNoContent {
		_ = json.NewDecoder(resp.Body).Decode(&out)
	}
	return resp, out
}

func (s *testServer) signUp(email string) string {
	s.t.Helper()
	creds := map[string]string{"email": email, "password": "password123"}
	if resp, body := s.do(http.MethodPost, "/api/v1/auth/register", "", creds); resp.StatusCode != http.StatusCreated {
		s.t.Fatalf("register: status %d, body %v", resp.StatusCode, body)
	}
	resp, body := s.do(http.MethodPost, "/api/v1/auth/login", "", creds)
	if resp.StatusCode != http.StatusOK {
		s.t.Fatalf("login: status %d, body %v", resp.StatusCode, body)
	}
	return body["access_token"].(string)
}

func TestHealthz(t *testing.T) {
	s := newTestServer(t)
	resp, body := s.do(http.MethodGet, "/healthz", "", nil)
	if resp.StatusCode != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("status %d, body %v", resp.StatusCode, body)
	}
	if resp.Header.Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID header")
	}
}

func TestAuthFlow(t *testing.T) {
	s := newTestServer(t)
	token := s.signUp("alice@example.com")

	resp, body := s.do(http.MethodGet, "/api/v1/me", token, nil)
	if resp.StatusCode != http.StatusOK || body["email"] != "alice@example.com" {
		t.Fatalf("me: status %d, body %v", resp.StatusCode, body)
	}
	if _, leaked := body["password_hash"]; leaked {
		t.Fatal("password hash leaked in response")
	}

	resp, _ = s.do(http.MethodPost, "/api/v1/auth/register", "",
		map[string]string{"email": "alice@example.com", "password": "password123"})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate register: status %d", resp.StatusCode)
	}

	resp, _ = s.do(http.MethodPost, "/api/v1/auth/login", "",
		map[string]string{"email": "alice@example.com", "password": "wrong-password"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login: status %d", resp.StatusCode)
	}
}

func TestAuthRequired(t *testing.T) {
	s := newTestServer(t)
	for _, token := range []string{"", "garbage"} {
		resp, _ := s.do(http.MethodGet, "/api/v1/tasks", token, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("token %q: status %d, want 401", token, resp.StatusCode)
		}
	}
}

func TestBadRequests(t *testing.T) {
	s := newTestServer(t)
	token := s.signUp("bob@example.com")

	tests := []struct {
		name   string
		method string
		path   string
		body   any
		want   int
	}{
		{"unknown field", http.MethodPost, "/api/v1/tasks", map[string]any{"title": "x", "foo": 1}, http.StatusBadRequest},
		{"validation", http.MethodPost, "/api/v1/tasks", map[string]any{"title": ""}, http.StatusUnprocessableEntity},
		{"bad id", http.MethodGet, "/api/v1/tasks/abc", nil, http.StatusBadRequest},
		{"missing task", http.MethodGet, "/api/v1/tasks/999", nil, http.StatusNotFound},
		{"bad limit", http.MethodGet, "/api/v1/tasks?limit=ten", nil, http.StatusBadRequest},
		{"bad status filter", http.MethodGet, "/api/v1/tasks?status=lost", nil, http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := s.do(tt.method, tt.path, token, tt.body)
			if resp.StatusCode != tt.want {
				t.Fatalf("status %d, want %d, body %v", resp.StatusCode, tt.want, body)
			}
		})
	}
}

func TestTaskCRUD(t *testing.T) {
	s := newTestServer(t)
	token := s.signUp("carol@example.com")

	resp, created := s.do(http.MethodPost, "/api/v1/tasks", token,
		map[string]any{"title": "Prepare for interview", "priority": 3})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status %d, body %v", resp.StatusCode, created)
	}
	if created["status"] != "todo" {
		t.Fatalf("default status = %v", created["status"])
	}
	path := "/api/v1/tasks/" + jsonID(created)

	resp, updated := s.do(http.MethodPatch, path, token, map[string]any{"status": "done"})
	if resp.StatusCode != http.StatusOK || updated["status"] != "done" || updated["title"] != "Prepare for interview" {
		t.Fatalf("update: status %d, body %v", resp.StatusCode, updated)
	}

	resp, list := s.do(http.MethodGet, "/api/v1/tasks?status=done", token, nil)
	if resp.StatusCode != http.StatusOK || list["total"].(float64) != 1 {
		t.Fatalf("list: status %d, body %v", resp.StatusCode, list)
	}

	resp, stats := s.do(http.MethodGet, "/api/v1/tasks/stats", token, nil)
	if resp.StatusCode != http.StatusOK || stats["done"].(float64) != 1 {
		t.Fatalf("stats: status %d, body %v", resp.StatusCode, stats)
	}

	if resp, _ := s.do(http.MethodDelete, path, token, nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: status %d", resp.StatusCode)
	}
	if resp, _ := s.do(http.MethodGet, path, token, nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get after delete: status %d", resp.StatusCode)
	}
}

func TestUsersCannotSeeEachOthersTasks(t *testing.T) {
	s := newTestServer(t)
	alice := s.signUp("alice@example.com")
	mallory := s.signUp("mallory@example.com")

	_, created := s.do(http.MethodPost, "/api/v1/tasks", alice, map[string]any{"title": "secret"})
	path := "/api/v1/tasks/" + jsonID(created)

	if resp, _ := s.do(http.MethodGet, path, mallory, nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("foreign get: status %d", resp.StatusCode)
	}
	if resp, _ := s.do(http.MethodDelete, path, mallory, nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("foreign delete: status %d", resp.StatusCode)
	}
	_, list := s.do(http.MethodGet, "/api/v1/tasks", mallory, nil)
	if list["total"].(float64) != 0 {
		t.Fatalf("foreign list: %v", list)
	}
}

type failingTasks struct {
	TaskService
}

func (failingTasks) Stats(context.Context, int64) (domain.TaskStats, error) {
	return domain.TaskStats{}, errors.New("db is down")
}

type stubTokens struct{}

func (stubTokens) Parse(string) (int64, error) { return 1, nil }

type downDB struct{}

func (downDB) Ping(context.Context) error { return errors.New("down") }

func TestInternalErrorsAreHidden(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewHandler(nil, failingTasks{}, stubTokens{}, downDB{}, log)
	srv := httptest.NewServer(h.Routes())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/tasks/stats", nil)
	req.Header.Set("Authorization", "Bearer x")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusInternalServerError || bytes.Contains(body, []byte("db is down")) {
		t.Fatalf("status %d, body %s", resp.StatusCode, body)
	}

	resp, err = http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("healthz status %d, want 503", resp.StatusCode)
	}
}

type panickingTasks struct {
	TaskService
}

func (panickingTasks) Stats(context.Context, int64) (domain.TaskStats, error) {
	panic("unexpected")
}

func TestPanicRecovery(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewHandler(nil, panickingTasks{}, stubTokens{}, nil, log)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/stats", nil)
	req.Header.Set("Authorization", "Bearer x")

	h.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", rec.Code)
	}
}

func jsonID(m map[string]any) string {
	b, _ := json.Marshal(m["id"])
	return string(b)
}
