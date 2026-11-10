.PHONY: up down restart logs ps hooks build test cover vet fmt fmt-check lint vuln tidy-check check

# Development runs in Docker via the root docker-compose.yml. After a code
# change, run `make up` again: it rebuilds the image and recreates the container.
COMPOSE := docker compose -f ../docker-compose.yml
SERVICE := backend

# Build the image and start the backend in the background; returns once the
# /health healthcheck passes (or fails if the container never becomes healthy).
up: .env
	$(COMPOSE) up --build --detach --wait $(SERVICE)

# Stop and remove the compose project's containers and network.
down:
	$(COMPOSE) down

restart: down up

# Follow the backend's JSON logs (Ctrl+C to stop following).
logs:
	$(COMPOSE) logs --follow $(SERVICE)

ps:
	$(COMPOSE) ps

# Enable the versioned git hooks (.githooks/pre-commit runs the same gate as
# `make check` + secret/artifact checks). Run once per clone.
hooks:
	git config core.hooksPath .githooks
	@echo "git hooks enabled: pre-commit (checks) and commit-msg (Conventional Commits)"

# Compose requires backend/.env; fail early with a hint instead of a stat error.
.env:
	@echo "backend/.env is missing. Create it with: cp .env.example .env" >&2
	@exit 1

# ---- Local tooling (no config or Docker needed) ----
# Dev tools are pinned in tools.mod (never in go.mod or the image).
TOOL := go tool -modfile=tools.mod

build:
	CGO_ENABLED=0 go build -o bin/server ./cmd/server

test:
	go test ./... -race -cover

cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out -o coverage.html

vet:
	go vet ./...

# Format in place: gofumpt + gci (import groups), configured in .golangci.yml.
fmt:
	$(TOOL) golangci-lint fmt ./...

# Fail (and show the diff) if anything is not formatted.
fmt-check:
	$(TOOL) golangci-lint fmt --diff ./...

lint:
	$(TOOL) golangci-lint run ./...

# Known vulnerabilities reachable from our code (Go vulnerability database).
vuln:
	$(TOOL) govulncheck ./...

# Fail if go.mod/go.sum are not tidy.
tidy-check:
	go mod tidy -diff

# The full gate run by the verifier before every commit. Cheapest checks first.
check: tidy-check fmt-check vet lint test vuln
	@echo "check: all passed"
