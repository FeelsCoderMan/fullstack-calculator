# Logging and error handling spec

Status: **agreed**. It applies to all backend code. `AGENTS.md` summarizes it, and this
document is the full reference. Code that disagrees with it is a bug.

## 1. Goals

1. **One log record per event, and nothing logged twice.** An HTTP request is one event
   and produces exactly one record. An error is part of the request that failed, so it
   goes into that request's record and is never logged on its own as well.
2. **Errors are logged once, where they're handled.** Lower layers *return* errors with
   context (`fmt.Errorf("...: %w", err)`). They never log them.
3. **Clients get codes and developers get causes.** The response carries only a stable
   error code. The frontend turns it into a user-friendly, localized message (AGENTS.md
   Decision 2). The log record carries the full cause chain, and a stack trace for
   unexpected failures.
4. **Production stays quiet, and debug can be switched on.** Levels are chosen so that
   `info` in production shows every request and every failure. `debug` adds detail for
   hard-to-reproduce cases. It's enabled with `LOG_LEVEL=debug` (in `backend/.env`)
   and a restart.

## 2. Who logs what

| Event                         | Logged by                               | Category | Record                                   |
|-------------------------------|-----------------------------------------|----------|------------------------------------------|
| HTTP request handled          | `RequestLogger` (`internal/middleware`) | `http`   | exactly one `request completed` record  |
| Error while handling a request | nobody separately; it goes into the request record | `http` | fields on `request completed` |
| Panic in a handler            | `RequestLogger` (recovers it)           | `http`   | the request record, as a 500 with a stack |
| Server lifecycle              | `internal/server`                       | `server` | `server listening`, `shutdown signal received, draining connections`, `server stopped` |
| Startup or run failure        | `cmd/server` (`main`)                   | `server` | one `server failed to start` / `server exited with error` |
| net/http internal errors      | `http.Server.ErrorLog` bridge           | `http`   | net/http's own text (accepted exception) |
| Diagnostic detail             | handler code, at `debug` only            | `http`   | see section 6                            |

Rules that follow from this table:

- **`service` and `repository` never log.** They return errors. `apperror` never logs.
- **Handlers never log errors.** A handler that fails calls `writeError(c, err)` and
  returns. `writeError` writes the JSON response and *attaches* the error to the request
  with `middleware.RecordFailure`. It doesn't print anything.
- **`RequestLogger` writes the single record** after the handler chain has finished, so
  it knows the final status, the duration and the attached error.
- A function that returns an error doesn't log it. A function that logs an error doesn't
  return it.

## 3. Request flow

```
request ─▶ RequestLogger ─▶ (gin routing) ─▶ handler ─▶ service
              │                                 │           │
              │                                 │      returns err (wrapped, %w)
              │                                 ▼
              │                       writeError(c, err)
              │                         ├─ apperror.From(err) → *apperror.Error (code, stack)
              │                         ├─ middleware.RecordFailure(c, {err, appErr, code})
              │                         └─ c.JSON(status(code), {"code": ...})
              ▼
   after c.Next(): one record ── level from status, plus code / error / stack if attached
```

`RequestLogger` lives in `internal/middleware`, which must not import `handler`.
`newEngine` (used by `NewRouter` and the handler tests) wires it as
`middleware.RequestLogger(accessLog, writeError, pathHealth)`: it
gets the handler's `writeError` to answer a recovered panic (so code → status stays mapped
in one place), and the paths whose successful requests are logged at `debug`.

`RequestLogger` also:

- assigns a **request ID**. It takes the incoming `X-Request-ID` if that is 1–64 characters
  of `[A-Za-z0-9-]`, and otherwise generates one with `crypto/rand.Text()` (26 base32
  characters, 128 bits of randomness). It echoes the ID in the
  `X-Request-ID` response header and puts it in the record. A user who reports a problem
  can quote it, and it finds the exact record in Loki.
- **recovers panics.** Every panic becomes `apperror.Panic(v)`: `error.internal` with the
  panic's stack. It is answered with `writeError` (a 500 JSON response) if nothing was written yet, and
  reported in the same single record. A panic never produces net/http's multi-line stack
  dump. `http.ErrAbortHandler` is handled the same way, and is then re-panicked so net/http
  can abort the connection as it expects.
- also covers gin's 404 and 405 fallbacks. `NoRoute` and `NoMethod` use `writeError` with
  `apperror.ErrNotFound` and `apperror.ErrMethodNotAllowed`.

## 4. The request record

Message: always `request completed`. Attributes, in this order:

| Key           | Type      | When              | Example                                  |
|---------------|-----------|-------------------|------------------------------------------|
| `method`      | string    | always            | `"POST"`                                 |
| `path`        | string    | always            | `"/api/v1/divide"`                       |
| `status`      | int       | always            | `422`                                    |
| `duration_ms` | float     | always            | `0.84`                                   |
| `request_id`  | string    | always            | `"AQ6QWA3Y7JP7OYWH33L4W2B3JC"`            |
| `code`        | string    | request failed    | `"error.calculation.division.by.zero"` (the code the client received) |
| `error`       | string    | request failed    | `"divide 1 by 0: error.calculation.division.by.zero"` |
| `stack`       | []string  | unexpected failure (`error.internal`) | `["service.(*Calculator).Divide service/calculator.go:42", ...]` |

"Request failed" means `writeError` ran, or a panic was recovered. Every 4xx/5xx response
from this service goes through one of those two paths. `stack` appears only on
`error.internal` records that captured one, from `apperror.Internal`, `apperror.Panic`, or
`From` on an unknown error. An unmapped sentinel code is answered and logged as
`error.internal` without a stack: sentinels never capture one. A panic after the response
was already written keeps that response's status, but the record is still `ERROR` with a
stack.

`service`, `logger`, `time`, `level` and `msg` are added by the handler setup in
`internal/logger`.

Examples (formatted here; on stdout each is one line):

```json
{"time":"…","level":"INFO","msg":"request completed","service":"calculator-backend","logger":"http",
 "method":"POST","path":"/api/v1/divide","status":200,"duration_ms":0.31,"request_id":"9f2c…"}

{"time":"…","level":"INFO","msg":"request completed","service":"calculator-backend","logger":"http",
 "method":"POST","path":"/api/v1/divide","status":422,"duration_ms":0.27,"request_id":"5b1a…",
 "code":"error.calculation.division.by.zero","error":"divide 1 by 0: error.calculation.division.by.zero"}

{"time":"…","level":"ERROR","msg":"request completed","service":"calculator-backend","logger":"http",
 "method":"POST","path":"/api/v1/power","status":500,"duration_ms":1.9,"request_id":"c07e…",
 "code":"error.internal","error":"power: compute result: unexpected state",
 "stack":["service.(*Calculator).Power service/calculator.go:77","handler.(*Calc).Power handler/calc.go:51", "…"]}
```

## 5. Levels

| Outcome                                  | Level   | Why                                                       |
|------------------------------------------|---------|-----------------------------------------------------------|
| 5xx (incl. recovered panics)             | `ERROR` | **always**: something is broken on our side                |
| 4xx                                      | `INFO`  | the client's input was wrong; the service works as designed. Visible, never alarming |
| 2xx / 3xx                                | `INFO`  | normal traffic                                           |
| 2xx/3xx on `/health`                     | `DEBUG` | container probes every 10s must not flood Loki            |
| diagnostic detail (section 6)            | `DEBUG` | off in production unless `LOG_LEVEL=debug`               |

`WARN` is reserved for server-level conditions that need attention but aren't failures,
for example `file logging enabled; development only`. It is never used per request.

## 6. Debug records

Debug records exist for cases that are hard to reproduce, such as which operand failed
validation and why, or what a parsed body looked like. They're the only exception to
"one record per request", and they follow strict rules:

1. Level `debug` only. They're invisible at the default `LOG_LEVEL=info`.
2. They add **new** information that isn't in the request record. They never repeat the
   status, error, code or stack.
3. They carry the `request_id` so they line up with the request record.
4. They come from handler code through one helper (`debugLog(c, msg, attrs...)`, added
   with its first use). `service` and `repository` still never log.
5. They never contain secrets or whole request bodies. Log the relevant fields with typed
   attributes.

Enabling: set `LOG_LEVEL=debug` in `backend/.env` and run `make up`, which recreates the
container. Set it back to `info` afterwards.

## 7. Errors: `internal/apperror`

`apperror` connects returned errors to codes, status codes and logs.

- **`Code`** is a typed string enum. The values are the API's error codes (AGENTS.md
  catalog), with top-level codes (`Codes()`) separate from field codes (`FieldCodes()`).
  They're mirrored as enums in `api/openapi.yaml`, and a test keeps the two identical.
- **`*Error`** holds `Code`, optional `Fields []FieldError` (validation), an optional
  wrapped `cause`, and an optional stack.
  - `Error()` is `code` or `code: cause`. It is developer text, never sent to clients.
  - `Unwrap()` returns the cause, so `errors.Is` and `errors.As` see through it.
  - `Is(target)` matches by code, so `errors.Is(err, apperror.ErrDivisionByZero)` works
    however the error was built or wrapped.
- **Sentinels** (`ErrNotFound`, `ErrDivisionByZero`, …) are for expected failures. They
  carry **no stack**, because the code and the wrap context explain them fully.
- **`Internal(err)`** is for unexpected failures. It wraps `err` as `error.internal` and
  **captures the stack at the call site**. Call it where the unexpected error is first
  seen, so the stack points at the real origin.
- **`Validation(fields...)`** builds `error.validation.failed` with per-field codes. That
  code is internal: it sets the 422 status and the log record's `code`, but the response
  body is only the array of field errors.
- **`Panic(v)`** builds `error.internal` from a recovered panic value, with the stack of
  the panic.
- **`From(err)`** finds the `*Error` in the chain (`errors.As`). If there is none, the error
  is unexpected and becomes `Internal(err)`. In that case the stack points at the handler,
  which is why origins should call `Internal` themselves.
- **`Stack()`** returns frames as `"<pkg>.<func> <dir>/<file>:<line>"`, for example
  `"service.(*Calculator).Divide service/calculator.go:42"`. There are at most 32 frames.
  Both the import path and the file path are cut to their last element, so local paths,
  such as a developer's home directory, never reach the logs.

Mapping a code to an HTTP status happens **only** in the handler (`statusFor`), in one
place. `apperror` knows nothing about HTTP.

## 8. Responses

- **Error body:** `{"code": "<code>"}`, except for a validation error (422), whose body is a
  bare array `[{"field": "...", "code": "..."}]` with no top-level code. No messages, no
  stack and no cause text. A 500 always answers just `{"code":"error.internal"}`.
- The user-friendly message is produced by the **frontend** from `code`, with a generic
  fallback for unknown codes.
- Every documented response's shape is in `api/openapi.yaml` (`ErrorResponse`,
  `ValidationErrorResponse`, `ErrorCode`, `FieldErrorCode`), and handler tests validate
  real responses against it with `assertMatchesSpec`.

## 9. Checklist for new code

- [ ] Lower layers return errors wrapped with `%w`, and never log them.
- [ ] Expected failures return an `apperror` sentinel, possibly wrapped with context.
- [ ] Unexpected failures are wrapped with `apperror.Internal(err)` at their origin.
- [ ] Handlers end every failure with `writeError(c, err)` and log nothing themselves.
- [ ] Any new code is added to `apperror`, `statusFor`, `api/openapi.yaml`, the frontend
      mapping and the AGENTS.md catalog in the same change.
- [ ] Debug records follow section 6.
