ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: run dev build test test-race audit fmt db-up migrate-up migrate-down migrate-storage migrate-storage-dry-run migration seed docs jobs frontend dev-all init test-integration frontend-install frontend-lint frontend-typecheck frontend-build docker-cloud-run docker-tools-cloud-run check
run:
	go run ./cmd/api
dev:
	go run github.com/air-verse/air@v1.67.4 -c .air.toml
frontend:
	cd frontend && npm run dev
dev-all: frontend dev
build:
	go build ./...
test:
	go test ./...
test-race:
	go test -race ./...
fmt:
	gofmt -w cmd internal
audit:
	go mod verify
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./...
	go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...
db-up:
	docker compose up -d postgres mailpit
migrate-up:
	go run ./cmd/migrate up
migrate-down:
	go run ./cmd/migrate down
migrate-storage-dry-run:
	go run ./cmd/migrate-storage --dry-run
migrate-storage:
	go run ./cmd/migrate-storage
migration:
	python3 scripts/migration.py "$(name)"
seed:
	@echo 'System roles are seeded by migrations. Use cmd/admin to bootstrap real verified users; see README.'
docs:
	go run ./cmd/api openapi > docs/openapi.json
jobs:
	go run ./cmd/jobs
init:
	python3 scripts/setup.py
test-integration:
	@TEST_DATABASE_URL="$(DATABASE_URL)" go test ./cmd/api
frontend-install:
	cd frontend && npm ci
frontend-lint:
	cd frontend && npm run lint
frontend-typecheck:
	cd frontend && npm run typecheck
frontend-build:
	cd frontend && npm run build
docker-cloud-run:
	docker buildx build --platform linux/amd64 --target api --load -t pmt-web-api:cloud-run .
docker-tools-cloud-run:
	docker buildx build --platform linux/amd64 --target tools --load -t pmt-web-tools:cloud-run .
check: fmt audit test test-race build frontend-lint frontend-typecheck frontend-build
