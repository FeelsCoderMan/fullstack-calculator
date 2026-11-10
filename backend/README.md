# Calculator backend

A Go REST service for the full-stack calculator. It uses [gin](https://github.com/gin-gonic/gin)
for routing, `log/slog` for structured JSON logs, and shuts down gracefully on SIGINT/SIGTERM.

> **Status:** the server, configuration, logging, health check and Docker setup are in place.
> The calculator endpoints (`/api/v1/...`) and the OpenAPI/Swagger docs are not implemented yet.

## Requirements

Install the tools for the way you want to run the server.

| Tool                   | Version                         | Needed for                                 |
|------------------------|---------------------------------|--------------------------------------------|
| [Go](https://go.dev/dl/) | **1.27.3** or newer (`go.mod`) | tests, vet, coverage, local builds         |
| Docker Engine          | 23+ (BuildKit on by default)    | running the server (development runs in Docker) |
| Docker Compose         | v2.24+ (for `env_file.required` and `up --wait`) | `make up` / `docker compose up` |
| GNU Make               | any                             | the `make` shortcuts                       |

Check what you have:

```sh
go version              # go version go1.27.3 ...
docker --version
docker compose version
```

You don't need to install anything else. Go modules download on the first build, and the
hot-reload tool [air](https://github.com/air-verse/air) is pinned in `tools.mod` and runs
through `go tool`, so don't install it globally.

## Configuration

All configuration comes from environment variables, and **every variable is required**.
There are no defaults in code. If a variable is missing or empty, the server exits at startup
with an error such as `missing environment variable LOG_DIR`.

Create your config file once, from the committed example:

```sh
cp .env.example .env    # run in backend/
```

`.env` is gitignored. Docker Compose loads it into the container and refuses to start
without it.

| Variable           | Example | Description                                                              |
|--------------------|---------|--------------------------------------------------------------------------|
| `PORT`             | `8080`  | Listen port, `0`–`65535` (`0` = random free port, tests only). Keep `8080` with Docker. |
| `SHUTDOWN_TIMEOUT` | `10s`   | How long in-flight requests get to finish on shutdown (Go duration, > 0). |
| `LOG_LEVEL`        | `info`  | `debug`, `info`, `warn` or `error` (offsets like `info+2` also accepted). |
| `LOG_TO_FILE`      | `false` | **Development only.** Also writes logs to per-category files in `LOG_DIR`. |
| `LOG_DIR`          | `logs`  | Directory for those log files. Required even when `LOG_TO_FILE=false`.    |

## Running the server

Development runs in **Docker**. Docker Compose builds the image from the root
`docker-compose.yml` and runs it with `backend/.env`. The server listens on
`http://localhost:8080`.

### With Make (from `backend/`)

```sh
cp .env.example .env    # once
make up                 # build the image, start the container, wait until /health is healthy
make logs               # follow the JSON logs (Ctrl+C stops following, not the server)
make down               # stop and remove the containers and network
```

There is no hot reload. **After changing code, run `make up` again.** It rebuilds the image
and recreates the container. `make restart` does a full `down` + `up`, and `make ps` shows the
container status. `make up` stops with a hint if `.env` is missing.

### With Docker Compose directly (from the project root)

```sh
cp backend/.env.example backend/.env   # once
docker compose up --build              # add -d to run in the background
docker compose down
```

The image is built in two stages: a static binary on `golang:1.27.3-alpine`, run on
`alpine:3.24` as a non-root user. The image contains no config. Compose injects
`backend/.env`, health-checks `/health` every 10s, and gives the server 15s to shut down
gracefully.

### Without Docker (optional)

To run the binary on your machine, export the variables yourself:

```sh
set -a; . ./.env; set +a
go run ./cmd/server
# or, with hot reload via the pinned air tool:
go tool -modfile=tools.mod air
```

### Check that it works

```sh
curl -i http://localhost:8080/health
# HTTP/1.1 200 OK
# Content-Type: application/json; charset=utf-8
#
# {"status":"ok"}
```

## Endpoints

| Method        | Path      | Response                 |
|---------------|-----------|--------------------------|
| `GET`, `HEAD` | `/health` | `200 {"status":"ok"}`    |

Unknown routes return `404`, and a wrong method on a known path returns `405`.

## Logging

The server writes one JSON object per line to **stdout**, ready for Grafana Loki. Every record
carries `service` and `logger` (`server` or `http`). Each request produces one access-log
record with `method`, `path`, `status` and `duration_ms`. Successful `/health` checks are
logged at `debug`, so health probes don't flood the logs.

```json
{"time":"2026-11-10T17:33:43Z","level":"INFO","msg":"server listening","service":"calculator-backend","logger":"server","addr":"[::]:8080"}
```

## Development

All commands run from `backend/`.

```sh
make test     # go test ./... -race -cover
make cover    # coverage.out + coverage.html
make vet      # go vet ./...
make fmt      # format in place: gofumpt + gci (import groups)
make lint     # golangci-lint run (config: .golangci.yml)
make vuln     # govulncheck: known vulnerabilities reachable from our code
make check    # everything a commit must pass: tidy, fmt, vet, lint, test, vuln
make build    # static binary at bin/server
```

The tools ([golangci-lint](https://golangci-lint.run) v2 and
[govulncheck](https://go.dev/doc/tutorial/govulncheck)) are pinned in `tools.mod` and
run through `go tool`, so there's nothing to install. The first run downloads and builds
them.

### Committing

Every commit has to pass `make check` plus secret and artifact checks on exactly the
staged changes. With Claude Code, use the `/commit` skill. It runs the `verifier`
subagent and, only on PASS, the `committer` subagent. A hook blocks any `git commit` that
hasn't been verified.

For commits made outside Claude Code (terminal, IDE), enable the versioned git hooks
once per clone:

```sh
make hooks    # git config core.hooksPath .githooks
```

From then on, two hooks run on every commit:

- `.githooks/pre-commit` runs the same gate, and skips it if the staged changes were
  already verified.
- `.githooks/commit-msg` requires a [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/)
  message: `<type>(<scope>)!: <summary>`, with type `feat`, `fix`, `refactor`, `perf`,
  `test`, `docs`, `style`, `build`, `ci`, `chore` or `revert`, an imperative lowercase
  summary of at most 72 characters with no trailing period, and a blank line before the
  body. A `!` must be paired with a `BREAKING CHANGE:` footer. The full rules are in
  `.claude/agents/committer.md`.

To run the gate yourself:

```sh
../.claude/scripts/verify.sh    # stage everything first; prints VERIFY: PASS/FAIL
```

These run locally with Go. They don't need `.env` or Docker, because the tests build their own
environment.

## Project layout

```
backend/
├── cmd/server/          # main: server.NewApp() then app.Run(), logs any error, exits 1
├── internal/
│   ├── server/          # App: loads config, sets up logging, listens, graceful shutdown
│   ├── handler/         # HTTP layer (gin): routes, handlers, access-log middleware
│   ├── config/          # env loading: config.LoadServer, config.LoadLog
│   └── logger/          # slog setup, categories, typed attributes, dev file output
├── .env.example         # config template (copy to .env)
├── Dockerfile
├── Makefile             # up/down/logs (Docker) + test/cover/vet/build
├── .air.toml            # optional local hot reload config
├── .golangci.yml        # linters + formatters (golangci-lint v2)
└── tools.mod            # dev tool pins (golangci-lint, govulncheck, air), kept out of go.mod
```

Project conventions and design decisions are in [`../AGENTS.md`](../AGENTS.md).
