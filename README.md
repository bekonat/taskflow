# TaskFlow

A REST API for personal task management written in Go. It stores data in PostgreSQL and authenticates users with JWT.

The project follows a layered architecture (transport → service → repository). Every layer depends on interfaces rather than concrete types, so each one can be unit-tested in isolation.

## Features

- User registration and login with bcrypt password hashing and JWT access tokens
- Task CRUD with ownership checks: users can only see and modify their own tasks
- Filtering by status, full-text search (`ILIKE` with escaping), and pagination
- Aggregated statistics computed in SQL with `FILTER` clauses
- Embedded SQL migrations applied automatically at startup
- Middleware for request IDs, structured logging (`log/slog`), and panic recovery
- Graceful shutdown on `SIGINT`/`SIGTERM`
- In-memory storage fallback when `DATABASE_URL` is not set
- Unit tests, HTTP tests using `httptest`, and integration tests against a real PostgreSQL
- Docker multi-stage build, docker-compose, and GitHub Actions CI

## Tech stack

Go 1.23 · `net/http` (Go 1.22+ routing patterns) · PostgreSQL 16 · pgx/v5 · golang-jwt · bcrypt · Docker

## Project structure

```
cmd/api/                 entrypoint, dependency wiring, graceful shutdown
internal/config/         configuration from environment variables
internal/domain/         entities, validation, domain errors
internal/service/        business logic, repository interfaces
internal/auth/           JWT manager and password hasher
internal/repository/
    postgres/            pgx-based repositories and migration runner
    memory/              thread-safe in-memory repositories
internal/transport/
    httpapi/             handlers, middleware, error mapping
migrations/              SQL migrations embedded into the binary
```

## Getting started

```bash
docker compose up --build
```

To run locally without Docker (this uses in-memory storage):

```bash
make run
```

## API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET    | `/healthz` | – | Liveness probe and DB ping |
| POST   | `/api/v1/auth/register` | – | Create an account |
| POST   | `/api/v1/auth/login` | – | Get an access token |
| GET    | `/api/v1/me` | ✓ | Current user |
| GET    | `/api/v1/tasks?status=&q=&limit=&offset=` | ✓ | List tasks |
| POST   | `/api/v1/tasks` | ✓ | Create a task |
| GET    | `/api/v1/tasks/{id}` | ✓ | Get a task |
| PATCH  | `/api/v1/tasks/{id}` | ✓ | Partially update a task |
| DELETE | `/api/v1/tasks/{id}` | ✓ | Delete a task |
| GET    | `/api/v1/tasks/stats` | ✓ | Task counts by status, plus overdue tasks |

### Example

```bash
curl -X POST localhost:8080/api/v1/auth/register \
  -d '{"email":"me@example.com","password":"password123"}'

TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/login \
  -d '{"email":"me@example.com","password":"password123"}' | jq -r .access_token)

curl -X POST localhost:8080/api/v1/tasks -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Prepare for interview","priority":3,"due_date":"2026-12-01T10:00:00Z"}'

curl "localhost:8080/api/v1/tasks?status=todo&q=interview" -H "Authorization: Bearer $TOKEN"
```

Errors are returned as JSON with a matching HTTP status code:

```json
{ "error": "must be between 1 and 3", "field": "priority" }
```

## Testing

```bash
make test
make up && make test-integration
make cover
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDR` | `:8080` | Listen address |
| `DATABASE_URL` | empty | PostgreSQL DSN; in-memory storage is used when it is empty |
| `JWT_SECRET` | – | HMAC secret, at least 16 characters (required) |
| `TOKEN_TTL` | `24h` | Access token lifetime |
