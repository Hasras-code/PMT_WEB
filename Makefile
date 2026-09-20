ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: run dev build test test-race audit fmt db-up migrate-up migrate-down migration seed docs jobs
run:
	go run ./cmd/api
dev:
	go run github.com/air-verse/air@v1.67.4 -c .air.toml
build:
	go build ./...
test:
	go test ./...
test-race:
	go test -race ./...
fmt:
	gofmt -w cmd internal integration
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
migration:
	python3 scripts/migration.py "$(name)"
seed:
	@echo 'System roles are seeded by migrations. Use cmd/admin to bootstrap real verified users; see README.'
docs:
	go run ./cmd/openapi > docs/openapi.json
jobs:
	go run ./cmd/jobs

.PHONY: init test-integration
init:
	python3 scripts/setup.py
test-integration:
	@TEST_DATABASE_URL="$(DATABASE_URL)" go test ./integration
