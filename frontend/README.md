# Calculator frontend

A single-page React + TypeScript app, built with [Vite](https://vite.dev), for the full-stack
calculator. It sends every calculation to the Go backend and shows the result, or an error
explained in plain language. There is no router: the whole app is one page.

It covers all seven operations of the backend: add, subtract, multiply, divide, power,
square root and percentage. Results can be chained like on a pocket calculator.

## Requirements

| Tool                     | Version                          | Needed for                                  |
|--------------------------|----------------------------------|---------------------------------------------|
| Docker Engine            | 23+ (BuildKit on by default)     | running the app (backend + frontend)        |
| Docker Compose           | v2.24+                           | `docker compose up`                         |
| [Node.js](https://nodejs.org) | **22.12** or newer (`engines`) | the dev server, tests, lint, local builds |
| npm                      | 10+ (ships with Node)            | installing dependencies, running scripts    |

Check what you have:

```sh
docker compose version
node --version          # v22.12.0 or newer
npm --version
```

## Running the app

The whole stack runs with **Docker Compose from the project root**. Compose builds both
images, starts the backend, waits until its `/health` check passes, and then starts the
frontend.

```sh
cp backend/.env.example backend/.env   # once: the backend refuses to start without it
docker compose up --build              # add -d to run in the background
```

| URL                                   | What                                         |
|---------------------------------------|----------------------------------------------|
| <http://localhost:3000>               | the calculator (frontend)                    |
| <http://localhost:3000/swagger/>      | Swagger UI, proxied to the backend           |
| <http://localhost:8080>               | the backend directly                         |

```sh
docker compose logs -f frontend   # nginx access log
docker compose down               # stop and remove everything
```

After a code change, run `docker compose up --build` again. It rebuilds the image and
recreates the container.

### How the Docker image works

`frontend/Dockerfile` has two stages:

1. **Build** on `node:22-alpine`: `npm ci`, then `npm run build` (type check + Vite build into `dist/`).
2. **Run** on `nginxinc/nginx-unprivileged:1.30-alpine`, as a non-root user on port 8080.
   [`nginx.conf`](nginx.conf) serves `dist/` and reverse-proxies `/api/` and `/swagger/` to
   the `backend` service.

Because nginx proxies the API, the browser only ever talks to one origin, so the backend
needs no CORS. Hashed files under `/assets/` are cached for a year, and `index.html` is
always revalidated, so a new deploy is picked up right away. Compose runs the container with
a read-only filesystem (nginx gets a `tmpfs` for `/tmp`) and health-checks it.

## Development server (hot reload)

For frontend work, run the backend in Docker and the frontend with Vite's dev server.
Changes then show up in the browser instantly.

```sh
# 1. Backend, from the project root (or: make -C backend up)
docker compose up --build --detach --wait backend

# 2. Frontend, from frontend/
npm install          # once, and after package.json changes
npm run dev          # http://localhost:5173
```

The dev server proxies `/api` and `/swagger` to `http://localhost:8080`
([`vite.config.ts`](vite.config.ts)), the same way nginx does in Docker. If the backend
isn't running, the UI says *"The calculator service is unavailable"*.

There's no `.env` for the frontend. It calls relative paths (`/api/v1/...`), so it has
nothing to configure.

## Commands

All commands run from `frontend/`.

```sh
npm run dev         # dev server with hot reload, http://localhost:5173
npm run build       # type check (tsc -b) + production build into dist/
npm run preview     # serve dist/ locally (no API proxy)
npm run typecheck   # tsc -b
npm run lint        # oxlint (config: .oxlintrc.json)
npm test            # vitest, once
npm run test:watch  # vitest, watch mode
npm run coverage    # tests + coverage report (text + coverage/index.html)
npm run gen:api     # regenerate src/api/schema.gen.ts from ../backend/api/openapi.yaml
npm run check       # typecheck + lint + tests
```

## Talking to the API

The contract is [`../backend/api/openapi.yaml`](../backend/api/openapi.yaml). The frontend
**generates** its TypeScript types from that file with
[openapi-typescript](https://openapi-ts.dev), so a change to the spec shows up as a type
error and can't drift silently.

```sh
npm run gen:api     # after every change to openapi.yaml; commit the result
```

`src/api/schema.gen.ts` is committed. The Docker build only sees `frontend/`, so it can't
read the backend's spec.

Every operation is one `POST`:

```
POST /api/v1/add        {"leftOperand": 1.5, "rightOperand": 2}                  → 200 {"result": 3.5}
POST /api/v1/divide     {"leftOperand": 1, "rightOperand": 3, "precision": 2}    → 200 {"result": 0.33}
POST /api/v1/divide     {"leftOperand": 1, "rightOperand": 0}                    → 422 {"code": "error.calculation.division.by.zero"}
POST /api/v1/sqrt       {"operand": 2.25}                                        → 200 {"result": 1.5}
POST /api/v1/power      {"leftOperand": 0, "rightOperand": -1}                   → 422 {"code": "error.calculation.zero.negative.power"}
POST /api/v1/subtract   {"rightOperand": "x"}                                    → 422 [{"field":"leftOperand","code":"error.field.required"},
                                                                                        {"field":"rightOperand","code":"error.field.invalid.type"}]
```

| Operation   | Path                  | Body                            | Inputs in the UI          | Math errors (422)                    |
|-------------|-----------------------|---------------------------------|---------------------------|--------------------------------------|
| Add         | `/api/v1/add`         | `leftOperand`, `rightOperand`   | first + second number     | `result.overflow`                    |
| Subtract    | `/api/v1/subtract`    | `leftOperand`, `rightOperand`   | number − amount           | `result.overflow`                    |
| Multiply    | `/api/v1/multiply`    | `leftOperand`, `rightOperand`   | first × second number     | `result.overflow`                    |
| Divide      | `/api/v1/divide`      | `leftOperand`, `rightOperand`   | dividend ÷ divisor        | `division.by.zero`, `result.overflow` |
| Power       | `/api/v1/power`       | `leftOperand`, `rightOperand`   | base ^ exponent           | `zero.negative.power`, `result.overflow`, `result.undefined` |
| Square root | `/api/v1/sqrt`        | `operand`                       | one number                | `negative.square.root`               |
| Percentage  | `/api/v1/percentage`  | `leftOperand`, `rightOperand`   | percent *of* a number     | `result.overflow`                    |

Math error codes are listed without their `error.calculation.` prefix. Each one has its own
message in `src/i18n/en.ts`, e.g. `0 ^ -1` shows *"Zero can’t be raised to a negative power:
that divides by zero."*
The authoritative list per operation is in the spec and in Swagger UI.

**Square root takes one operand.** Its body is `{"operand", "precision"?}`
(`UnaryOperationRequest` in the spec). `operand` is any number, decimals included (`2.25` →
`1.5`), so a decimal result can carry straight into a square root. The client doesn't reject
negative numbers: they are valid input, and the backend answers
`error.calculation.negative.square.root`, shown as *"A negative number has no real square root."*

### Continuing from a result

The calculate key is **=**, and it works like a pocket calculator. After a successful
calculation, the result carries over into the first number. The second number is cleared and
focused, ready for the next operand:

```
15 + 12  =  27      → first number becomes 27, cursor in the second number
pick ×, type 2, =   → 27 × 2 = 54
```

The display keeps showing the finished calculation (`15 + 12 = 27`) until the next one, even
while you pick another operation. For square root, the result replaces the single number. A
failed calculation leaves the inputs as they were, so you can fix them. Pressing *Enter*, or
typing `=` in a number field, also calculates. The carried value is the exact result
(`0.30000000000000004`, `1e+21`), so nothing is lost between steps.

Every operation also takes an optional **Decimal places** setting (`precision`, 0–15). Left on
*Full precision*, the field is not sent and the backend returns the full float64 result
(e.g. `0.1 + 0.2 = 0.30000000000000004`).

### Errors

The backend sends codes, never text. The frontend turns every code into a message from
`src/i18n`:

| Response                                   | What the user sees                                         |
|--------------------------------------------|------------------------------------------------------------|
| `200 {"result"}`                           | the expression and the result in the display               |
| `422 [ {field, code}, ... ]`               | a message under each matching input (`aria-invalid`, focus moves to the first one) |
| `4xx/5xx {"code"}`                         | a message in the display, plus the `X-Request-ID` as a reference |
| unknown code                               | a generic message (*"Something went wrong"*)               |
| network failure (no response)              | *"Can't reach the server"* (`error.client.network`)         |
| non-JSON 5xx (proxy error page)            | *"The calculator service is unavailable"* (`error.client.unavailable`) |
| any other unexpected body                  | *"The server sent an unexpected response"*                 |

The `error.client.*` codes exist only in the frontend, for failures that never reach the API.

The client validates input **before** sending, with the backend's own field codes: empty →
`error.field.required`, not a plain decimal number (`abc`, `1,5`, `0x10`, `Infinity`) →
`error.field.invalid.type`, beyond float64 (`1e309`) → `error.field.out.of.range`. The
backend is still the source of truth: it validates again, and its field errors land under the
same inputs.

Only the latest request counts. Submitting again (or switching operation, or *Clear*) aborts
the request in flight, so a slow earlier answer can never overwrite a newer result.

## Responsive layout and accessibility

**540px and below counts as mobile.** The layout is plain CSS (CSS Modules, flex and grid,
`rem`, `clamp()`), with one breakpoint, `@media (max-width: 540px)`:

| Element          | Desktop (> 540px)                          | Mobile (≤ 540px)                                |
|------------------|--------------------------------------------|--------------------------------------------------|
| Operations       | one row of 7 tiles                         | two rows (4 + 3) of larger tap targets           |
| Operands         | `left  op  right` on one line              | stacked, with the operator between them          |
| Precision + actions | side by side                            | stacked; full-width buttons, *Calculate* first   |
| Result display   | large monospace, wraps long numbers        | same, smaller padding                            |

The page has no horizontal scroll down to 320px wide. Inputs use a font size of at least 16px,
so iOS doesn't zoom into a focused field. Light and dark mode follow the system setting
(`prefers-color-scheme`), and animations stop under `prefers-reduced-motion`.

Accessibility:

- Every input has a `<label>`. The operation picker is a native radio group (`fieldset` +
  `legend`), so the arrow keys move between operations.
- Field errors set `aria-invalid` and are linked with `aria-describedby`. After a failed
  submit, focus moves to the first invalid field.
- The result display is an `<output aria-live="polite">`, so screen readers announce each
  result or error. It's marked `aria-busy` while a request is in flight.
- Everything works by keyboard: *Enter* in any field, or `=` in a number field, calculates.
  Focus rings are always visible. The **=** button's accessible name is *Calculate*.

## Conventions

- **No hardcoded strings.** User-facing text comes only from `src/i18n` (`en.ts`). API paths,
  field names, error codes, headers and other values are `as const` objects in
  `src/api/constants.ts`, typed against the generated spec types. Class names come from CSS
  Modules (`styles.x`), not string literals.
- **Components never call `fetch`.** They go through `src/api/client.ts`, which returns a
  typed outcome (`success` / `fieldErrors` / `failure`) and never throws for HTTP or network
  errors.
- **Strict TypeScript** (`strict`, `noUncheckedIndexedAccess`), no `any`.
- **Tests** are table-driven (`it.each` with named rows), following
  `.claude/skills/table-driven-tests/SKILL.md`. User flows (type, click, wait) are
  straight-line tests. Component tests mock `src/api/client`, and the client is tested with a
  mocked `fetch`.
- TypeScript is pinned to **5.9** because `openapi-typescript` requires TypeScript 5.

## Project layout

```
frontend/
├── src/
│   ├── api/
│   │   ├── schema.gen.ts     # generated from the OpenAPI spec (npm run gen:api), committed
│   │   ├── types.ts          # aliases for the generated types
│   │   ├── constants.ts      # operations, paths, field names, error codes, headers
│   │   ├── operations.ts     # arity, inputs per operation, request building, field mapping
│   │   └── client.ts         # calculate(): fetch + response classification
│   ├── i18n/                 # every user-facing text, code → message lookups
│   ├── hooks/useCalculator.ts # form state, validation, request lifecycle, result carry-over
│   ├── components/           # Calculator, OperationPicker, Field, ResultDisplay (+ CSS Modules)
│   ├── utils/                # input validation, class name / aria id helpers
│   ├── test/setup.ts         # jest-dom matchers, cleanup between tests
│   ├── App.tsx               # page shell: header, calculator, footer link to Swagger UI
│   ├── index.css             # design tokens (light/dark) and base styles
│   └── main.tsx              # entry point
├── index.html
├── nginx.conf                # static files + /api and /swagger reverse proxy (Docker)
├── Dockerfile                # node build → nginx-unprivileged runtime
├── vite.config.ts            # dev proxy + Vitest config
├── .oxlintrc.json            # lint rules (react, a11y, typescript, vitest)
└── package.json
```

Project conventions and design decisions are in [`../AGENTS.md`](../AGENTS.md). The backend
is described in [`../backend/README.md`](../backend/README.md).
