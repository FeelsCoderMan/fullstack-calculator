# Full-stack calculator

A calculator with a **React + TypeScript** frontend and a **Go** REST microservice. The
frontend sends every calculation to the backend, which does the arithmetic and answers with
the result or an error code. The frontend turns that code into a message.

Operations: add, subtract, multiply, divide, power, square root and percentage, each with
optional rounding (`precision`, 0–15 decimal places).

```
browser ──▶ frontend (nginx, :3000) ──/api, /swagger──▶ backend (Go + gin, :8080)
              serves the React app                        does the math, serves Swagger UI
```

This README covers running **both parts together**. Each part has its own README with the
details: [`backend/README.md`](backend/README.md) and [`frontend/README.md`](frontend/README.md).

## Requirements

To run the app you only need Docker:

| Tool           | Version                                          |
|----------------|--------------------------------------------------|
| Docker Engine  | 23+ (BuildKit on by default)                     |
| Docker Compose | v2.24+ (for `env_file.required` and `up --wait`) |
| GNU Make       | any (optional: the `make` shortcuts; plain `docker compose` works too) |

To run the tests and coverage reports on your machine, you also need
[Go](https://go.dev/dl/) **1.27.4+** (backend) and [Node.js](https://nodejs.org) **22.12+**
with npm 10+ (frontend).

## Quick start: build and run everything

From the project root, with the root `Makefile`:

```sh
make env    # once: creates backend/.env from the example (the backend has no defaults)
make up     # builds both images, starts backend, then frontend; returns once both are healthy
```

`make up` runs `docker compose up --build --detach --wait`: Compose builds both images,
starts the backend, waits until its `/health` check passes, then starts the frontend. It
prints the URLs when the stack is ready (`make urls` prints them again). The ports in
those URLs are read from the running stack, so `ports:` in `docker-compose.yml` is the only
place they are defined. `make help` lists every target.

Without Make, the same with plain Compose (in the foreground):

```sh
cp backend/.env.example backend/.env
docker compose up --build
```

| URL                                | What                                            |
|------------------------------------|-------------------------------------------------|
| <http://localhost:3000>            | **the calculator**                              |
| <http://localhost:3000/swagger/>   | Swagger UI (the API docs, proxied to the backend) |
| <http://localhost:8080>            | the backend directly (e.g. `/health`, `/api/v1/...`) |

```sh
make ps              # both services should say (healthy)
make logs            # follow the logs of both (backend logs are JSON lines)
make logs-backend    # or one service: logs-backend, logs-frontend
make down            # stop and remove everything
make restart         # down + up
```

After a code change in either part, run `make up` again. It rebuilds what changed and
recreates the containers.

### Check that it works

```sh
curl -s -H 'Content-Type: application/json' -d '{"leftOperand":1.5,"rightOperand":2}' \
  http://localhost:3000/api/v1/add
# {"result":3.5}
```

## Tests and coverage

Tests run locally, without Docker or `.env`.

| | Backend (from `backend/`) | Frontend (from `frontend/`) |
|---|---|---|
| Install | nothing (Go modules and tools download on first run) | `npm install` |
| Tests | `make test` (race detector, coverage summary) | `npm test` |
| Coverage report | `make cover` → `coverage.html` | `npm run coverage` → `coverage/index.html` |
| Everything a commit must pass | `make check` (format, vet, lint, tests, vulnerabilities, tidy) | `npm run check` (types, lint, tests) |

## API at a glance

Every operation is a `POST` under `/api/v1` with a JSON body. The full contract is the
OpenAPI spec, [`backend/api/openapi.yaml`](backend/api/openapi.yaml), also served as Swagger
UI.

```
POST /api/v1/add      {"leftOperand": 1.5, "rightOperand": 2}                → 200 {"result": 3.5}
POST /api/v1/divide   {"leftOperand": 1, "rightOperand": 3, "precision": 2}  → 200 {"result": 0.33}
POST /api/v1/divide   {"leftOperand": 1, "rightOperand": 0}                  → 422 {"code": "error.calculation.division.by.zero"}
POST /api/v1/sqrt     {"operand": 2.25}                                      → 200 {"result": 1.5}
POST /api/v1/add      {"leftOperand": "x"}                                   → 422 [{"field":"leftOperand","code":"error.field.invalid.type"},
                                                                                    {"field":"rightOperand","code":"error.field.required"}]
```

`subtract`, `multiply`, `power` and `percentage` take the same body as `add`. The backend
documents every endpoint with examples in [`backend/README.md`](backend/README.md#endpoints).

## Design decisions

- **The backend sends data or codes, never messages.** Success is `{"result": ...}`, errors
  are `{"code": "error...."}`, and field validation errors are an array of
  `{"field", "code"}` listing every invalid field at once. All user-facing text lives in the
  frontend (`frontend/src/i18n`), so wording and language change without touching the API.
- **Spec first.** `backend/api/openapi.yaml` is the single source of truth. The backend's
  tests check every response against it, and the frontend generates its TypeScript types
  from it.
- **Lightweight clean architecture** in the backend: `handler → service`, with HTTP (gin)
  confined to `handler` and `middleware`, and the math in `service`. There's no repository,
  because the calculator stores nothing.
- **float64 numbers**, with the limits made explicit: overflow, `NaN` and division by zero
  are errors, never `Inf`/`NaN` in a response. The reasoning is in
  [`backend/docs/operations.md`](backend/docs/operations.md).
- **One origin.** nginx in the frontend image serves the app and proxies `/api` and
  `/swagger` to the backend, so the browser needs no CORS.
- **Structured logs** for Grafana Loki: one JSON record per request
  ([`backend/docs/logging.md`](backend/docs/logging.md)).

All agreed conventions and decisions are in [`AGENTS.md`](AGENTS.md).

## Repository layout

```
.
├── Makefile             # make up / down / logs / ps: the whole stack through docker compose
├── docker-compose.yml   # runs backend + frontend together
├── backend/             # Go service: API, OpenAPI spec, Swagger UI  → backend/README.md
├── frontend/            # React + Vite app, served by nginx in Docker → frontend/README.md
├── AGENTS.md            # conventions and design decisions (CLAUDE.md imports it)
├── PROMPTS.md           # the AI prompts used to build this project
└── .githooks/           # pre-commit and commit-msg hooks (enable: make -C backend hooks)
```

This project was built with Claude Code. Every prompt is listed in [`PROMPTS.md`](PROMPTS.md).
