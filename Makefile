# Runs the whole stack (backend + frontend) with the root docker-compose.yml, and
# the unit tests and coverage reports of both parts. Per-part tooling lives in
# backend/Makefile (Go) and frontend/package.json (npm); these targets call it.
.PHONY: help env up down restart build urls logs logs-backend logs-frontend ps \
	test test-backend test-frontend coverage coverage-backend coverage-frontend

COMPOSE := docker compose -f docker-compose.yml
BACKEND_DIR := backend
FRONTEND_DIR := frontend
ENV_FILE := $(BACKEND_DIR)/.env
ENV_EXAMPLE := $(BACKEND_DIR)/.env.example
NPM := npm --prefix $(FRONTEND_DIR)

# Where the coverage targets write their HTML reports.
BACKEND_COVERAGE_REPORT := $(BACKEND_DIR)/coverage.html
FRONTEND_COVERAGE_REPORT := $(FRONTEND_DIR)/coverage/index.html

# URL of a running service, from the host port Compose actually published for it
# (the `ports:` mapping in docker-compose.yml), so the Makefile never repeats it:
# $(call service_url,frontend) -> http://localhost:3000. Expanded by the shell
# when the recipe runs, after the stack is up.
service_url = http://localhost:$$($(COMPOSE) ps --format '{{(index .Publishers 0).PublishedPort}}' $(1))

help: ## List the targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{ printf "  make %-18s %s\n", $$1, $$2 }'

env: ## Create backend/.env from the example (never overwrites an existing one)
	@if [ -f $(ENV_FILE) ]; then echo "$(ENV_FILE) already exists, left as is"; \
	else cp $(ENV_EXAMPLE) $(ENV_FILE) && echo "created $(ENV_FILE) from $(ENV_EXAMPLE)"; fi

up: $(ENV_FILE) ## Build both images and start the stack; returns once both are healthy
	$(COMPOSE) up --build --detach --wait
	@$(MAKE) --no-print-directory urls

down: ## Stop and remove the containers and network
	$(COMPOSE) down

# Sequential sub-makes on purpose: as prerequisites, `make -j` could run down and
# up at once. backend/.env is checked first, so a missing file stops nothing.
restart: $(ENV_FILE) ## Recreate the stack from scratch
	$(MAKE) down
	$(MAKE) up

build: $(ENV_FILE) ## Build both images without starting them
	$(COMPOSE) build

urls: ## Print the URLs of the running stack (ports read from Compose)
	@if [ -z "$$($(COMPOSE) ps --quiet --status running frontend)" ]; then \
		echo "the stack is not running; start it with: make up" >&2; exit 1; fi
	@echo "calculator: $(call service_url,frontend)"
	@echo "API docs:   $(call service_url,frontend)/swagger/"
	@echo "backend:    $(call service_url,backend)"

logs: ## Follow the logs of both services (Ctrl+C stops following, not the stack)
	$(COMPOSE) logs --follow

logs-backend: ## Follow the backend's JSON logs
	$(COMPOSE) logs --follow backend

logs-frontend: ## Follow the frontend's nginx logs
	$(COMPOSE) logs --follow frontend

ps: ## Show container status and health
	$(COMPOSE) ps

# ---- Unit tests and coverage ----
# They run on this machine, not in Docker: Go for the backend, Node for the frontend
# (see README "Requirements"). No .env or running stack is needed. The two parts run
# one after the other, so their output never interleaves under `make -j`.

test: ## Run the unit tests of both parts (backend with -race and per-package coverage)
	@$(MAKE) --no-print-directory test-backend
	@$(MAKE) --no-print-directory test-frontend

test-backend: ## Run the backend unit tests (go test -race -cover)
	@echo "==> backend unit tests"
	$(MAKE) --no-print-directory -C $(BACKEND_DIR) test

test-frontend: $(FRONTEND_DIR)/node_modules ## Run the frontend unit tests (vitest)
	@echo "==> frontend unit tests"
	$(NPM) test

coverage: ## Run both test suites with coverage and print both coverage reports
	@$(MAKE) --no-print-directory coverage-backend
	@$(MAKE) --no-print-directory coverage-frontend
	@echo "==> HTML reports: $(BACKEND_COVERAGE_REPORT), $(FRONTEND_COVERAGE_REPORT)"

coverage-backend: ## Backend coverage: per-package and total, plus an HTML report
	@echo "==> backend coverage"
	$(MAKE) --no-print-directory -C $(BACKEND_DIR) cover

coverage-frontend: $(FRONTEND_DIR)/node_modules ## Frontend coverage: per-file table plus an HTML report
	@echo "==> frontend coverage"
	$(NPM) run coverage

# Installs the frontend's locked dependencies on first use (and when the lock file
# changes), so the npm targets work on a fresh clone.
$(FRONTEND_DIR)/node_modules: $(FRONTEND_DIR)/package-lock.json
	$(NPM) ci

# Compose requires backend/.env (the backend has no defaults); fail early with a
# hint instead of a compose error.
$(ENV_FILE):
	@echo "$(ENV_FILE) is missing. Create it with: make env (or cp $(ENV_EXAMPLE) $(ENV_FILE))" >&2
	@exit 1
