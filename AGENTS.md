# AGENTS.md

Guidance for AI coding agents (and humans) working in this repository. The rules in
**Decisions** are agreed with the project owner. Do not deviate from them without asking.

## Project

A full-stack calculator: a **React + TypeScript** frontend that calls a **Go** REST
microservice to perform arithmetic. This is a take-home assignment with a ~2–4 hour
budget. **Prioritize correctness, clarity, and maintainability over extra features.**

### Scope

- Required operations: `add`, `subtract`, `multiply`, `divide`
- Optional operations: `power`, `sqrt`, `percentage`
- Frontend: intuitive input/result UI, client-side validation, error display, basic mobile support
- Backend: REST endpoints, input validation, edge cases (division by zero, invalid JSON,
  missing operands, non-finite results), JSON responses, OpenAPI spec + Swagger UI
- Unit tests for both layers + a coverage report
- README: setup, how to run each side, API examples, design decisions/assumptions
- Docker: one Dockerfile per app + `docker-compose.yml` at the root

Do not add features beyond this scope (no auth, persistence, history, expression parser)
unless explicitly asked.

## Decisions

1. **Backend first, then frontend.** The backend, including its OpenAPI spec and tests, is
   finished and reviewed before frontend work starts. The frontend is built against the spec.
2. **Responses carry data or codes, never human-readable messages.** All user-facing text
   (success and error) is produced by the frontend from codes. See [Response format](#response-format).
3. **Clean architecture, lightweight:** `handler → service → repository`. No heavy DDD
   (no aggregates, value-object zoo, domain events, CQRS). See [Backend](#backend-go).
4. **Spec-first OpenAPI.** `backend/api/openapi.yaml` is the single source of truth for the
   contract. Swagger UI is served by the backend. The frontend generates its TS types from it.
5. **Docker:** `backend/Dockerfile`, `frontend/Dockerfile`, and a root `docker-compose.yml`
   that runs both together.
6. **No hardcoded strings.** Every string value in application code comes from an enum
   (typed constants) or a helper function. See [No hardcoded strings](#no-hardcoded-strings).
7. **Structured logging** with `log/slog`, JSON lines on stdout (shipped to Grafana Loki).
   Optional dev-only file output. See [Logging](#logging-backend).
8. **Gin for HTTP routing**, confined to `internal/handler`. `http.Server` (timeouts, graceful
   shutdown) stays in `internal/server`. See [Rules](#rules).

## No hardcoded strings

String literals are allowed **only in the declaration of a constant, enum, or sentinel
error**, and those declarations live in dedicated files (see below). Everything else
references the name or calls a helper that builds the value.

- **Go**:
  - Use a typed string enum for closed sets, e.g. `type Code string` with
    `const CodeDivisionByZero Code = "error.calculation.division.by.zero"`, or
    `type Category string`.
  - Use an untyped `const` block for one-off values: env var names, defaults, routes and
    paths, header names and values, log messages, and log attribute keys.
  - Use helpers for derived values, e.g. `logger.FileName(category, t)`.
  - **Error text is not enumerated.** Messages passed to `fmt.Errorf` are written inline
    at the call site, e.g. `fmt.Errorf("listen on %s: %w", addr, err)`. They are developer
    context for logs, not API output, and are most readable next to the code that fails.
    Identity comes from the wrapped sentinel (`%w` + `errors.Is`), never from the text.
    The API never exposes this text, only codes (see [Response format](#response-format)).
- **TypeScript**:
  - Use `as const` objects or string-literal unions, e.g. API paths, field names and codes
    from the generated OpenAPI types, CSS class names, storage keys.
  - User-facing text comes **only** from `src/i18n`.
- **Allowed exceptions**:
  - format strings written directly in a `fmt` formatting call (`fmt.Errorf`, `fmt.Sprintf`),
    e.g. `fmt.Sprintf("%s %s:%d", fn, file, line)`. A format string is most readable next to
    its arguments; a constant may still be used when it is reused or names a contract.
  - struct tags (`json:"a"`)
  - the empty string `""` as a zero-value check (`if v != ""`)
  - test files asserting **exact wire output** (expected JSON bodies, status payloads). The
    literal there is deliberate: it catches an accidental change to the constant itself.
  - config files (`.air.toml`, Dockerfiles, YAML)

### Where enums, constants, and errors live (Go)

Each package keeps its declarations in two dedicated files, the same way a domain keeps
`ErrInvalidEmail` in its own `errors.go`:

| File           | Contains                                                                                   |
|----------------|--------------------------------------------------------------------------------------------|
| `errors.go`    | sentinel errors (`var ErrInvalidEnv = errors.New(...)`) and typed error structs            |
| `constants.go` | typed enums + their helpers (e.g. `Categories()`), const blocks: env names, defaults, paths, headers, log messages, attribute keys |

- No `const` blocks or `errors.New` sentinels in any other file of the package. Local
  variables are fine.
- Create a file only when the package has something to put in it (no empty `errors.go`).
- Group related constants under a comment header (`// Log messages.`, `// URL paths ...`).
- `apperror` is the exception by design: it *is* the error package, so its codes and
  errors are its main content (`codes.go`, `errors.go`).

## Repository layout

```
.
├── AGENTS.md                 # this file (CLAUDE.md imports it)
├── README.md
├── PROMPTS.md                # AI prompts used (assignment deliverable)
├── docker-compose.yml
├── .githooks/pre-commit      # git hook: runs .claude/scripts/verify.sh (make -C backend hooks)
├── backend/                  # Go module
│   ├── Dockerfile
│   ├── Makefile              # up/down/logs (Docker dev) + fmt/lint/vuln/test/check shortcuts
│   ├── .golangci.yml         # golangci-lint v2: linters + formatters (gofumpt, gci)
│   ├── .air.toml             # optional local hot reload config (not part of the dev workflow)
│   ├── tools.mod             # dev tool pins (golangci-lint, govulncheck, air), separate from go.mod
│   ├── api/openapi.yaml      # contract, embedded into the binary
│   ├── docs/logging.md       # logging + error handling spec (one record per request)
│   ├── cmd/server/           # main: server.NewApp() + app.Run(), logs either error, exits 1
│   └── internal/
│       ├── server/           # App: config, wiring (DI), listen, graceful shutdown
│       ├── handler/          # HTTP layer: routing, decode/validate, DTOs, error→status mapping, middleware, swagger
│       ├── service/          # business logic (calculator operations), no net/http or gin imports
│       ├── repository/       # persistence ports/adapters (none needed yet; see below)
│       ├── apperror/         # error code catalog + typed errors shared across layers
│       ├── config/           # env-based configuration
│       └── logger/           # slog setup, categories, typed attrs, dev-only file output
└── frontend/                 # Vite + React + TypeScript
    ├── Dockerfile
    └── src/
        ├── api/              # typed API client + generated OpenAPI types
        ├── i18n/             # code → message mapping (success + error texts)
        ├── components/       # presentational components
        ├── hooks/            # stateful logic (e.g. useCalculator)
        └── utils/            # validation helpers
```

## Response format

### Success (2xx): data only

No `message`, no `code`, no envelope.

```json
{ "result": 2.5 }
```

### Error (4xx/5xx): code only

```json
{ "code": "error.calculation.division.by.zero" }
```

### Validation error (422): a bare array of field codes

No envelope, no top-level `code`: the status says the request failed, and each entry says
which field and why. All field problems are reported at once, not just the first one, in
a stable order (the request's fields in schema order, then unknown properties
alphabetically).

```json
[
  { "field": "leftOperand", "code": "error.field.required" },
  { "field": "rightOperand", "code": "error.field.invalid.type" }
]
```

`field` is the JSON property name as sent by the client. A 422 body is either this array
or a `{code}` object (math errors); the frontend tells them apart with `Array.isArray`,
maps `field` to its input and `code` to a localized message. It must have a generic
fallback for unknown codes.

### Error code conventions

- Lowercase, dot-separated, prefixed with `error.`: `error.<area>.<reason...>`
  (e.g. `error.user.invalid.character`).
- Codes are **API contract**: never rename one without updating the spec, frontend mapping,
  and tests together. Add new codes instead of overloading existing ones.
- All codes are defined as constants in `backend/internal/apperror` (`codes.go`: `Codes()` for
  top-level codes, `FieldCodes()` for field codes) and listed as the `ErrorCode` /
  `FieldErrorCode` enums in `openapi.yaml`. `TestCodesMatchSpec` keeps them identical. No
  string literals for codes elsewhere.
- `error.validation.failed` is **internal**: it marks an error that carries field errors,
  selects the 422 status and is the `code` of the request log record, but it is never sent
  (so it is not in `Codes()` or the spec).
- Code → HTTP status is mapped in exactly one place: `codeStatus` in
  `internal/handler/response.go` (`TestEveryCodeHasAStatus` catches a missing entry).
- Expected failures return an `apperror` sentinel (`ErrDivisionByZero`, …), optionally wrapped
  with context, and are matched with `errors.Is` by code. Unexpected failures are wrapped
  with `apperror.Internal(err)` at their origin, which records the stack. A 500 response is
  always just `{"code":"error.internal"}`. See `backend/docs/logging.md` section 7.

### Error code catalog

| Code                                     | HTTP | When                                              |
|------------------------------------------|------|---------------------------------------------------|
| `error.request.invalid.json`             | 400  | body is not valid JSON / not a JSON object        |
| `error.validation.failed`                | 422  | internal only: body is the bare field-error array  |
| `error.field.required`                   | —    | field-level: operand missing or `null`             |
| `error.field.invalid.type`               | —    | field-level: not a JSON number (`precision`: not an integer) |
| `error.field.out.of.range`               | —    | field-level: not representable as float64 (`precision`: outside 0–15) |
| `error.field.unknown`                    | —    | field-level: unexpected property in body           |
| `error.request.not.found`                | 404  | unknown route                                      |
| `error.request.method.not.allowed`       | 405  | wrong HTTP method                                  |
| `error.request.body.too.large`           | 413  | body exceeds limit (1 MiB)                         |
| `error.request.unsupported.media.type`   | 415  | `Content-Type` not `application/json`              |
| `error.calculation.division.by.zero`     | 422  | `divide` with `b == 0`                             |
| `error.calculation.negative.square.root` | 422  | `sqrt` with `a < 0`                                |
| `error.calculation.result.overflow`      | 422  | result is ±Inf                                     |
| `error.calculation.result.undefined`     | 422  | result is NaN (e.g. `power(-8, 0.5)`)              |
| `error.internal`                         | 500  | unexpected failure (details only in server logs)   |

Rule of thumb: **400** = the body can't be read as a JSON object; **422** = it can be read,
but a field or the math is invalid. Gin's default 404/405 plain-text responses (`NoRoute`/`NoMethod`) must be replaced
with the JSON codes above.

## API endpoints

Base path `/api/v1`, JSON bodies. The full definitions are in `backend/api/openapi.yaml`.

Design notes and rationale for the operations (number type, limits, rounding, error
shapes) are in [`backend/docs/operations.md`](backend/docs/operations.md).

Two-operand operations take `{"leftOperand": number, "rightOperand": number, "precision"?:
integer 0–15}` (shown as `L`, `R` below). `precision` rounds the result half away from
zero; omitted or `null` means the full float64 result. Results are always finite and never
`-0`. Implemented so far: `add`, `subtract`, `multiply`. `sqrt` gets its own request shape, designed separately.

| Method | Path                    | Body                          | Result                 |
|--------|-------------------------|-------------------------------|------------------------|
| POST   | `/api/v1/add`           | `{L, R, precision?}`          | `L + R`                |
| POST   | `/api/v1/subtract`      | `{L, R, precision?}`          | `L - R`                |
| POST   | `/api/v1/multiply`      | `{L, R, precision?}`          | `L * R`                |
| POST   | `/api/v1/divide`        | `{L, R, precision?}`          | `L / R`                |
| POST   | `/api/v1/power`         | `{L, R, precision?}`          | `L ^ R`                |
| POST   | `/api/v1/sqrt`          | to be designed                | `√x`                   |
| POST   | `/api/v1/percentage`    | `{L, R, precision?}`          | `L` percent of `R`     |
| GET, HEAD | `/health`           | none                          | `204`, no body         |
| GET    | `/swagger/`             | none                          | Swagger UI             |
| GET    | `/swagger/openapi.yaml` | none                          | raw spec               |

## Backend (Go)

### Layers

```
HTTP request → handler → service → repository
```

- **handler**: the only layer that knows HTTP and the only package that imports gin. It decodes and validates the request shape,
  calls the service, writes the response, and maps `apperror` → status code in **one** place.
  It depends on a service **interface** declared in the handler package (consumer-side
  interfaces, idiomatic Go).
- **service**: pure business logic. No `net/http`, no gin, no JSON tags. It returns typed errors from
  `apperror` (e.g. `apperror.ErrDivisionByZero`). It is fully unit-testable without HTTP.
- **repository**: data access behind interfaces declared by the service. The calculator is
  stateless, so **no repository is implemented yet**. Do not create an empty or fake one just
  to fill the layer. Add it only when there is real state (e.g. calculation history), behind
  an interface the service owns.
- Dependencies point inward only: `handler → service → repository`; never the reverse.
  `apperror` may be imported by all layers. Wiring happens in `internal/server`.

### Rules

- Routing with [gin](https://github.com/gin-gonic/gin), confined to `internal/handler`.
  `handler.NewRouter` returns an `http.Handler`, so `internal/server` stays framework-agnostic
  and keeps owning `http.Server` (timeouts, graceful shutdown). Don't use `engine.Run()`.
  - Build the engine with `gin.New()`, never `gin.Default()`. Gin's logger and recovery
    middleware write plain text. `RequestLogger` writes the request records and recovers
    panics itself (`backend/docs/logging.md`).
  - `gin.SetMode(gin.ReleaseMode)`, so gin's debug output never pollutes the JSON log stream.
  - `HandleMethodNotAllowed = true`, so a wrong method returns 405, not 404.
- Other dependencies need a reason (e.g. test-only OpenAPI validation).
- `float64` arithmetic. Never return `NaN`/`±Inf`; map them to the codes above.
- Distinguish a missing operand from `0`. To report **all** field errors, decode the body into
  `map[string]json.RawMessage` and validate each field (required / type / range / unknown).
  Don't rely on `json.Decoder` errors, which stop at the first problem.
- Limit body size with `http.MaxBytesReader`. Reject trailing data after the JSON object.
- Never put internal error details in responses. Log them server-side (see [Logging](#logging-backend)).
- Config via env vars in `internal/config`, one loader per concern (`config.LoadServer`,
  `config.LoadLog`) so a caller reads only the variables it uses. A new concern gets its own
  struct + `LoadX(getenv)`. **Every variable is required. There are no defaults in code.** A
  missing or empty value fails startup with `config.ErrMissingEnv`. Values live in
  `backend/.env` (gitignored, created from the committed `backend/.env.example`), which compose
  requires (`make up` checks for it). The Dockerfile sets no `ENV`.

  | Env var            | Example | Purpose                                                |
  |--------------------|---------|--------------------------------------------------------|
  | `PORT`             | `8080`  | listen port (`0` = random, tests only)                 |
  | `SHUTDOWN_TIMEOUT` | `10s`   | graceful drain window                                  |
  | `LOG_LEVEL`        | `info`  | `debug` \| `info` \| `warn` \| `error` (slog offsets like `info+2` also accepted) |
  | `LOG_TO_FILE`      | `false` | **dev only**: also write logs to files in `LOG_DIR`    |
  | `LOG_DIR`          | `logs`  | directory for file logs (relative to the working dir)  |
  | `ALLOWED_ORIGINS`  | —       | CORS origins, for local dev (later step)               |
- Set server timeouts (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`) and
  shut down gracefully on SIGINT/SIGTERM.
- `make check` clean: formatted (gofumpt + gci), `go vet`, `golangci-lint` (`.golangci.yml`),
  race tests, `govulncheck`, tidy `go.mod`. Return errors, don't panic. A `//nolint` needs
  a specific linter and a reason. Fix a vulnerable dependency by upgrading it, not by ignoring it.

### Idiomatic Go

Write Go the way the standard library does. The references are
[Effective Go](https://go.dev/doc/effective_go),
[Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) and the
[Google Go Style Guide](https://google.github.io/styleguide/go/). If one of these conflicts
with a rule in [Decisions](#decisions) (e.g. no hardcoded strings), the Decision wins.

- **Naming:** packages are short, lowercase, single words with no `util`/`common`/`helpers`.
  Names don't repeat the package (`server.NewApp`, not `server.NewServerApp`; `config.Load`,
  not `config.LoadConfig`). Use MixedCaps and keep initialisms in one case (`ID`, `URL`,
  `HTTP`, `JSON`). Getters have no `Get` prefix (`app.Addr()`). Receiver names are one or two
  letters and the same on every method of a type. Keep names short when their scope is short.
- **Doc comments:** every exported identifier has one, and it starts with the identifier's
  name (`// NewApp loads ...`). Every package has a `// Package x ...` comment in one file.
- **Errors:** error strings are lowercase with no trailing punctuation and describe the
  failed action (`"listen on %s: %w"`). Wrap with `%w` to add context, and check with
  `errors.Is`/`errors.As`. Handle the error and return early, so the happy path stays at the
  left edge with no `else` after a `return`. Don't ignore errors silently. If you discard
  one on purpose, write `_ =` and a comment saying why.
- **Interfaces:** accept interfaces and return concrete types. Declare an interface in the
  package that consumes it, keep it small, and add it only when there is a second
  implementation or a test needs a fake. Don't create an interface "for the future."
- **Constructors and zero values:** use `NewX(...) (*X, error)` when setup can fail. Make
  zero values useful where that's cheap. Avoid `init()`, package-level mutable state, and
  globals. Pass dependencies in explicitly instead.
- **Context:** `ctx context.Context` is the first parameter of anything that blocks or does
  I/O. Don't store it in a struct. Use `context.Background()` only at the top (main, tests).
- **Concurrency:** every goroutine has a clear way to exit (ctx cancellation or a closed
  channel). Whoever starts a goroutine is responsible for stopping it. Prefer channels for
  handing off work and `sync.Mutex` for protecting state. All tests run with `-race`.
- **`main` stays thin:** it builds the app and runs it (`server.NewApp()`, then
  `app.Run(ctx)`), logs the one error that comes back, and exits non-zero. Wiring, listening
  and shutdown live in `internal/server`, where tests can reach them. Only `main` may call
  `os.Exit`.
- **Keep it simple:** use the standard library first. Avoid unnecessary abstraction, generics
  without a second use, and `interface{}`/`any` where a concrete type works. Prefer clear
  code over clever code.

### Logging (backend)

**Full spec: [`backend/docs/logging.md`](backend/docs/logging.md).** Read it before writing
code that handles errors or logs. The core rules:

- **One record per event, and nothing logged twice.** Each HTTP request produces exactly one
  `request completed` record, written by the `RequestLogger` middleware (category `http`)
  after the handler chain. It holds method, path, status, duration_ms and request_id, plus
  `code`, `error` and `stack` when the request failed.
- **Errors are logged once, in the request record.** `service` and `repository` return
  errors wrapped with `%w` and never log. Handlers end every failure with `writeError(c, err)`,
  which resolves the error through `apperror`, attaches it to the request for
  `RequestLogger` and writes the JSON `{code}` response. Handlers never log errors themselves.
- **Levels:** 5xx and recovered panics are always `ERROR` (with stack). 4xx is `INFO` (with
  code and cause). Successful `/health` is `DEBUG`. Other `DEBUG` records are diagnostic
  detail only (spec section 6), enabled with `LOG_LEVEL=debug` and a restart.
- `RequestLogger` assigns `X-Request-ID` (reusing a well-formed incoming one) and recovers
  panics into a 500 `error.internal`, reported in the same single record.

General rules:

- `log/slog` with `slog.JSONHandler` only. Never use `fmt.Print*` or the `log` package for
  application logs. One JSON object per line on **stdout**: this is what Loki ingests
  (via Promtail/Alloy or the Docker log driver). Don't log multi-line text.
- Loggers come from `internal/logger` (`logger.Factory`). Every logger has a **category**
  from the `logger.Category` enum (e.g. `server`, `http`), emitted as the `logger` attribute.
  Every record also carries `service`. Keep label-worthy values low-cardinality
  (`service`, `logger`, `level`). Put high-cardinality data (paths, durations, IDs) in
  attributes, never in the message.
- Messages are short, constant, past/present-tense phrases (`"server listening"`). Variable
  data goes in attributes, never interpolated into the message.
- **Typed attributes only.** Log with `(*slog.Logger).LogAttrs(ctx, level, msg, attrs...)`
  and build every attribute with a typed constructor from `internal/logger/attrs.go`
  (`logger.Addr(...)`, `logger.Status(...)`, `logger.Duration(...)`, `logger.Err(err)`, ...).
  - Never use the loose key/value form (`l.Info(msg, "key", v)`) or `With("k", v)`. It
    isn't type-checked: a missing value or a wrong key compiles fine.
  - Attribute keys are unexported in `logger`. A new field means a new constructor in
    `attrs.go` (+ a row in `attrs_test.go`), so keys and value types stay consistent for
    Loki queries.
  - Name logger variables by role (`serverLog`, `accessLog`), never `logger`, which would
    shadow the package.
- **File output (development only):** with `LOG_TO_FILE=true`, every record also goes to
  `LOG_DIR/<category>-<start timestamp>.log` (UTC, e.g. `logs/http-20261110T191503Z.log`), one
  file per category, alongside stdout. Files are closed on shutdown. `logs/` is gitignored.
  Never enable this in Docker/compose. Containers log to stdout only.
- Outside requests, log errors once at the boundary that handles them (`main` for startup
  and run failures). Never log an error and also return it.
- Exception: `http.Server.ErrorLog` (bridged via `slog.NewLogLogger`) puts net/http's own
  variable text in `msg`. This is accepted: it's rare and can't be avoided without a custom handler.

### OpenAPI / Swagger

- `backend/api/openapi.yaml` (OpenAPI 3.0.x) is written by hand, spec-first, and embedded
  with `//go:embed` by package `backend/api` (`api.Spec`, `api.SwaggerUI`). Served at
  `/swagger/openapi.yaml` (`application/yaml`) and as a Swagger UI page at `/swagger/`.
- **Workflow for a new endpoint:** first describe it in `openapi.yaml` (path, request body,
  every response status with its schema and error codes), then implement the handler, then
  test it. The frontend generates its types from the spec, so the spec is the deliverable
  that unblocks frontend work.
- Every endpoint documents its success schema and every possible error response,
  including the error code enum. A response with no body (e.g. `/health` → `204`)
  documents no `content`.
- Swagger UI is loaded from a pinned `swagger-ui-dist` release on jsDelivr with Subresource
  Integrity hashes (`api/swagger.html`). When upgrading it, recompute both `sha384` hashes
  and cross-check them against a second CDN.
- Spec tests (in `internal/handler`):
  - `TestRoutesMatchSpec`: every route registered in the router (`gin.Engine.Routes()`),
    except the docs routes, is documented in the spec, and vice versa.
  - `assertMatchesSpec(t, req, rec)`: validates a real response against the spec (the status
    must be documented, and the body must match its schema or be empty when none is
    declared). Call it in every handler test row for a documented response.
  - The spec is parsed and validated with `kin-openapi`, a test-only dependency.

### Tests

- **All test style rules live in the `table-driven-tests` skill**
  (`.claude/skills/table-driven-tests/SKILL.md`). Follow it, and don't duplicate its rules here.
- Coverage expectations: service tests cover every operation and every error branch.
  Handler tests use `httptest` and assert the **status code and exact JSON body** (including
  field-error array contents and order) for success, each error code, and multi-field validation.

### Development workflow (Docker)

- **Development runs in Docker.** `make up` (from `backend/`) builds the image from the root
  `docker-compose.yml`, starts the `backend` service, and waits until `/health` is healthy.
  After a code change, run `make up` again. It rebuilds and recreates the container. There is
  no hot reload in Docker. `make down` stops and removes the compose project.
- Tests, vet, and coverage run locally with Go (`make test`, `make vet`, `make cover`). They
  don't need `.env` or Docker.

### Hot reload (optional, local only)

- [air](https://github.com/air-verse/air) is pinned in `backend/tools.mod`, a separate
  modfile, so it **never** appears in `go.mod`, the production binary, or the Docker image.
  It is not part of the Makefile workflow. To use it, run
  `set -a; . ./.env; set +a; go tool -modfile=tools.mod air`. Don't install it globally or add
  it to `go.mod`. Upgrade it with `go get -modfile=tools.mod -tool github.com/air-verse/air@<version>`.
- `backend/.air.toml` sends SIGINT on reload, so the graceful shutdown path runs on every
  rebuild. Build output goes to `backend/tmp/` (gitignored).
- Air must not be referenced from Dockerfiles, compose, or CI.

Commands (run from `backend/`; see `Makefile`):

```sh
make up      # build image + start backend in Docker, wait until healthy (needs .env)
make down    # stop and remove the compose project
make restart # down + up
make logs    # follow backend logs
make ps      # container status
make build   # static binary at bin/server
make test    # go test ./... -race -cover
make cover   # coverage.out + coverage.html
make vet     # go vet
make fmt     # format in place (gofumpt + gci via golangci-lint)
make lint    # golangci-lint run
make vuln    # govulncheck
make check   # the full pre-commit gate: tidy-check, fmt-check, vet, lint, test, vuln
```

Tools are pinned in `backend/tools.mod` and run via `go tool -modfile=tools.mod <tool>`,
so they never appear in `go.mod` or the image. Upgrade one with
`go get -modfile=tools.mod -tool <module>@<version>`.

## Frontend (React + TypeScript)

Starts only after the backend is done (Decision 1).

- Vite, React, TypeScript with `strict: true`. No `any` without a comment justifying it.
- API types are **generated** from `backend/api/openapi.yaml` with `openapi-typescript`
  (`npm run gen:api`). Don't hand-write types that duplicate the spec.
- Components never call `fetch` directly. They go through `src/api`.
- All user-facing text comes from `src/i18n`: success messages are built client-side, and
  error messages map from `code` (top-level and per-field). Field errors render next to the
  matching input. Unknown codes fall back to a generic message.
- Validate client-side before sending (empty, non-numeric, missing operand) using the
  **same field codes** as the backend, but the backend stays the source of truth.
- Handle loading, network failure, and non-JSON responses gracefully.
- Accessibility: labelled inputs, named buttons, results/errors in an `aria-live` region,
  `aria-invalid` + `aria-describedby` on invalid fields, keyboard usable.
- Responsive via plain CSS (flex/grid, relative units).
- Vitest + React Testing Library + `@testing-library/user-event`. Test style follows the
  `table-driven-tests` skill (`it.each` with named rows).

Commands (run from `frontend/`):

```sh
npm install
npm run gen:api
npm run dev            # proxies /api to http://localhost:8080
npm test
npm run coverage
npm run lint
npm run build
```

## Docker

- `backend/Dockerfile`: multi-stage, builds a static binary (`CGO_ENABLED=0`), runs it in a
  minimal image (distroless/alpine) as a non-root user.
- `frontend/Dockerfile`: multi-stage, Node builds the app, nginx serves `dist/` and
  reverse-proxies `/api` (and `/swagger`) to the backend service, so the browser sees
  one origin and needs no CORS.
- Root `docker-compose.yml` runs both (`backend`, `frontend`) with a healthcheck on
  `/health`, and `frontend` waits for `backend` to be healthy.
  After `cp backend/.env.example backend/.env`, `docker compose up --build` must be the only
  command needed. Compose fails fast if `backend/.env` is missing.

## Working agreements for agents

1. **Plan, then implement in small steps:** backend service → handler + validation →
   OpenAPI + Swagger → backend Docker → frontend (client/types → i18n → UI) → frontend
   Docker → compose → README.
2. **Tests accompany code.** Delegate test writing to the `test-writer` subagent
   (`.claude/agents/test-writer.md`), or write them yourself following the
   `table-driven-tests` skill. Run the relevant suite before declaring a step done, and
   report real output. Never claim tests pass without running them.
3. **Review before finishing a step.** Use the `code-reviewer` subagent
   (`.claude/agents/code-reviewer.md`). It is read-only, so apply its findings yourself and
   note which ones you skipped and why.
4. A contract change (endpoint, field, error code) updates `openapi.yaml`, the `apperror`
   catalog above, the handlers, the tests, the frontend mapping, and the README in the same change.
5. Record AI prompts used for the work in `PROMPTS.md`.
6. Don't commit generated artifacts (`node_modules/`, `dist/`, `coverage/`, `coverage.out`,
   binaries). Commit only when the user asks.
7. **Every commit goes through verifier → committer** (the `commit` skill,
   `.claude/skills/commit/SKILL.md`):
   - The `verifier` subagent (`.claude/agents/verifier.md`) runs `.claude/scripts/verify.sh`
     on exactly the staged changes. That covers secret/artifact checks and
     `make -C backend check`.
   - Only on `VERDICT: PASS` does the `committer` subagent (`.claude/agents/committer.md`)
     write a Conventional Commit.
   - A PreToolUse hook (`.claude/hooks/require-verified-commit.sh`) blocks any
     `git commit` whose staged tree doesn't match the verified one, and blocks
     `--no-verify`. Never work around it. Fix the failure and verify again.
   - The versioned git hook `.githooks/pre-commit` runs the same `verify.sh` for every
     commit, including ones from a terminal, an IDE, or a script that the Claude hook can't
     see. It skips the checks if the staged tree was already verified. Enable it once per
     clone with `make -C backend hooks` (`git config core.hooksPath .githooks`).
   - Commit messages follow Conventional Commits. The rules (types, scopes, subject and
     body format, breaking changes) are in `.claude/agents/committer.md`, and the git hook
     `.githooks/commit-msg` enforces them on every commit. Keep the scope list in both files
     identical.
