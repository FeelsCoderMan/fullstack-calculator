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

Project is implemented in Linux/Fedora distribution and Claude Opus 5.5 is used.

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

Tests run locally, without Docker or `.env` (Go and Node.js, see
[Requirements](#requirements)). From the project root, one command runs both parts:

```sh
make test       # unit tests of both parts: backend (race detector, per-package coverage), then frontend
make coverage   # both suites with coverage, printing both reports and writing the HTML versions
```

`make coverage` prints the backend's coverage per package and in total, then the frontend's
per-file table (statements, branches, functions, lines, uncovered lines), and ends with the
paths of the HTML reports:

```
==> backend coverage
ok   fullstack-calculator/backend/internal/handler   coverage: 95.7% of statements
…
total:                                  (statements)   94.6%
==> frontend coverage
 Test Files  6 passed (6)
      Tests  157 passed (157)
File               | % Stmts | % Branch | % Funcs | % Lines | Uncovered Line #s
All files          |   98.36 |    98.38 |    98.7 |   98.64 |
…
==> HTML reports: backend/coverage.html, frontend/coverage/index.html
```

Each part also runs on its own: `make test-backend`, `make test-frontend`,
`make coverage-backend`, `make coverage-frontend`. The frontend targets run `npm ci` first
when `frontend/node_modules` is missing or older than `package-lock.json`. The parts run one
after the other, so their output never interleaves.

The same commands from inside each part:

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
- **Structured logs**: one JSON record per request on stdout
  ([`backend/docs/logging.md`](backend/docs/logging.md)).

All agreed conventions and decisions are in [`AGENTS.md`](AGENTS.md).

## Assumptions

What the project takes as given, where the assignment left it open:

- **One operation at a time.** A calculation is two numbers and an operator (one number
  for square root). There is no expression parser and no operator precedence; instead the
  result carries over into the next calculation, like on a pocket calculator.
- **Real numbers in float64.** Operands and results are 64-bit floats, the type the
  browser uses anyway:
  - integers above 2⁵³ lose precision (`9007199254740993` → `9007199254740992`), which is
    accepted, not an error;
  - values too small to represent (below about 2.5e-324, e.g. `1e-400`) become `0`;
  - operands beyond ±1.797e308 are rejected as out of range, and a result that overflows
    is an error;
  - `-0` is always returned as `0`.
- **Real results only.** No complex numbers: the square root of a negative number, and a
  negative base with a fractional exponent (`(-8) ^ 0.5`), are errors. So is `0` raised to
  a negative power, which would divide by zero.
- **Percentage means "L percent of R"**: `20% of 50 = 10`, i.e. `L × R / 100`. It is not a
  percentage change.
- **Rounding is optional.** By default the full float64 result is returned. `precision`
  (0–15 decimal places) rounds half away from zero, as on paper: `2.675` → `2.68`.
- **Stateless and anonymous.** No accounts, authentication, persistence or calculation
  history, so there is no database and no repository layer.
- **English only**, but every user-facing text lives in `frontend/src/i18n`, so another
  language means adding a file, not changing components.
- **Local deployment.** The stack runs on one machine with Docker Compose, on ports 3000 and
  8080, over plain HTTP. There is no TLS, rate limiting or production hardening beyond
  request size limits (1 MiB bodies, JSON only), timeouts and non-root containers.
- **Modern browsers.** The UI uses current CSS (`:has()`, `clip-path`) and supports widths
  down to 320px, with one mobile breakpoint at 540px. Swagger UI loads from a CDN, so the
  docs page needs internet access; the calculator itself does not.

## Repository layout

```
.
├── Makefile             # make up / down / logs / ps (whole stack via compose), make test / coverage (both parts)
├── docker-compose.yml   # runs backend + frontend together
├── backend/             # Go service: API, OpenAPI spec, Swagger UI  → backend/README.md
├── frontend/            # React + Vite app, served by nginx in Docker → frontend/README.md
├── AGENTS.md            # conventions and design decisions (CLAUDE.md imports it)
├── PROMPTS.md           # the AI prompts used to build this project
└── .githooks/           # pre-commit and commit-msg hooks (enable: make -C backend hooks)
```

This project was built with Claude Code. Every prompt is listed in [`PROMPTS.md`](PROMPTS.md).
