TEST_DATABASE_URL ?= postgres://taskflow:taskflow@localhost:5432/taskflow?sslmode=disable

.PHONY: run build test test-integration cover lint up down

run:
	JWT_SECRET=local-dev-secret-key go run ./cmd/api

build:
	go build -o bin/taskflow ./cmd/api

test:
	go test -race ./...

test-integration:
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -race -tags integration ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	go vet ./...
	gofmt -l .

up:
	docker compose up --build -d

down:
	docker compose down
