# Runs the whole stack (backend + frontend) with the root docker-compose.yml.
# Per-part tooling lives in backend/Makefile (Go) and frontend/package.json (npm).
.PHONY: help env up down restart build urls logs logs-backend logs-frontend ps

COMPOSE := docker compose -f docker-compose.yml
ENV_FILE := backend/.env
ENV_EXAMPLE := backend/.env.example

# URL of a running service, from the host port Compose actually published for it
# (the `ports:` mapping in docker-compose.yml), so the Makefile never repeats it:
# $(call service_url,frontend) -> http://localhost:3000. Expanded by the shell
# when the recipe runs, after the stack is up.
service_url = http://localhost:$$($(COMPOSE) ps --format '{{(index .Publishers 0).PublishedPort}}' $(1))

help: ## List the targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F ':.*## ' '{ printf "  make %-14s %s\n", $$1, $$2 }'

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

# Compose requires backend/.env (the backend has no defaults); fail early with a
# hint instead of a compose error.
$(ENV_FILE):
	@echo "$(ENV_FILE) is missing. Create it with: make env (or cp $(ENV_EXAMPLE) $(ENV_FILE))" >&2
	@exit 1
