# Operation endpoints: design notes

Status: **implemented for `add`, `subtract` and `multiply`.** The normative rules are in `AGENTS.md` and
`api/openapi.yaml`; this file keeps the decisions and their rationale for the remaining
operations to follow.

## 1. Request fields

Binary operations take `leftOperand` and `rightOperand` (replacing `a` / `b` in AGENTS.md).

```json
{ "leftOperand": 1.5, "rightOperand": 2 }
```

An optional `precision` field controls rounding (see section 4).

```json
{ "leftOperand": 1, "rightOperand": 3, "precision": 2 }
```

`field` in a field error is the JSON property name as sent: `leftOperand`, `rightOperand`,
`precision`.

`sqrt` is not covered here. It gets its own endpoint and request shape, designed separately
(noted so far: it takes a positive integer).

## 2. Number type: `float64`

- Operands and results are `float64` (range ±1.797e308).
- `int64` was rejected. It can't represent decimals (`divide` and `percentage` need
  them), and the browser parses every JSON number as a float64 anyway. An int64 result
  above 2⁵³ would arrive wrong in the UI (`9007199254740993` → `...992`).
- Arbitrary precision (`math/big`, numbers sent as strings) was rejected as out of scope.

## 3. Limits, overflow and float behaviour

| Situation | Detection | Response |
|---|---|---|
| Operand too large, e.g. `1e309` / `-1e309` | `strconv.ParseFloat` returns `±Inf` + `strconv.ErrRange` | 422, field code `error.field.out.of.range` |
| Result too large, e.g. `1.7e308 + 1.7e308`, `-1.7e308 - 1.7e308` | `math.IsInf(result, 0)` (covers both signs) | 422 `error.calculation.result.overflow` |
| Result NaN | `math.IsNaN(result)` | 422 `error.calculation.result.undefined` |
| Precision loss: integers above 2⁵³, e.g. `9007199254740993` → `9007199254740992`, `94906267 * 94906267` → `…288` (exact `…289`) | none | **accepted**, not an error |
| Underflow: magnitude below ~2.5e-324 (half the smallest positive float64, `5e-324`; values just above it round up to `5e-324`), e.g. input `1e-400` or result `1e-200 * 1e-200` (two non-zero operands) | none | **accepted, becomes `0`**, not an error |
| Subnormal result: magnitude between ~2.2e-308 and ~5e-324, e.g. `1e-160 * 1e-160` = `1e-320` | none | **accepted**, kept with fewer significant digits |
| Negative zero, e.g. `0 * -5`, `0 / -5`, `-0 + -0`, `-0 - 0`, input `-0`, `-1e-200 * 1e-200` | `result == 0` | **normalized to `0`**: the API never returns `-0` |

The precision-loss and underflow limits are documented in the API description in
`openapi.yaml` (and the README), so clients know about them.

A shared service helper (`finish`) runs on every operation's result, in this order:
reject Inf/NaN, round to `precision` if given, then turn `-0` into `0`. Normalizing last also
catches rounding that produces `-0` (e.g. `-0.001` at precision 2).

## 4. Field validation

Decode the body into `map[string]json.RawMessage`, check every field, and report all
problems at once, in a fixed order (`leftOperand`, `rightOperand`, `precision`, then
unknown keys).

| Raw value | Field code |
|---|---|
| missing or `null` | `error.field.required` |
| `"5"`, `true`, `{}`, `[]` | `error.field.invalid.type` |
| number outside float64 range | `error.field.out.of.range` |
| any other property | `error.field.unknown` |

### `precision` (optional)

- Number of decimal places to round the result to. Missing or `null` means no rounding
  (full float64 result). Integer from 0 to 15 (float64 holds ~15–17 significant digits).
- Validated with the operands, so its errors land in the same 422 array. No new codes:

  | Raw value | Field code |
  |---|---|
  | `"2"`, `true`, `2.5` (not an integer) | `error.field.invalid.type` |
  | `-1`, `16` | `error.field.out.of.range` |

- Rounding is **half away from zero on the shortest decimal form** of the result, as a
  person would round on paper: `2.675` → `2.68`, `1.005` → `1.01`, `2.5` → `3`,
  `-2.5` → `-3`. Implementation: `strconv.FormatFloat(r, 'g', -1, 64)` (shortest digits
  that round-trip), parsed into a `big.Rat`, rounded with `Rat.FloatString(places)`.
  Exact rational arithmetic, so it can't overflow, unlike `math.Round(r*10^p)/10^p`.
- Rejected: `strconv.FormatFloat(r, 'f', places, 64)`. It rounds the exact binary value
  half to even, so `2.5` → `2`, `0.5` → `0` (banker's rounding) and `2.675` → `2.67`
  (stored as 2.67499…). Found by a failing test while implementing `add`.
- Rounding happens in the service, so it is unit-tested with every operation.

## 5. Error bodies and statuses

**Changed from AGENTS.md:** field validation errors are **422** (was 400) and the body is a
bare JSON array of field errors, with no envelope: no top-level `code` and no `errors` key.
Every other error keeps the `{ "code": ... }` object.

| Status | When | Body |
|---|---|---|
| 400 | body is not valid JSON or not a JSON object | `{ "code": "error.request.invalid.json" }` |
| 422 | one or more field errors (`required`, `invalid.type`, `out.of.range`, `unknown`) | array of `{ field, code }`, all fields at once |
| 422 | fields are valid but the math is invalid (division by zero, overflow, ...) | `{ "code": ... }` |
| 404, 405, 413, 415, 500 | unchanged | `{ "code": ... }` |

New rule of thumb: **400** = the body can't be read as a JSON object; **422** = it can be
read, but its content is invalid (a field, or the math).

```json
[
  { "field": "leftOperand", "code": "error.field.out.of.range" },
  { "field": "rightOperand", "code": "error.field.invalid.type" }
]
```

What the implementation has to handle:

- **422 has two body shapes.** In the spec, the 422 response is
  `oneOf: [ErrorResponse, ValidationErrorResponse]`, where `ValidationErrorResponse` is
  `type: array, minItems: 1, items: FieldError`. The frontend tells them apart with
  `Array.isArray(body)`: an array maps to the inputs, an object to a form-level message.
- **Accepted trade-off:** a bare array can't gain sibling fields later (e.g. a trace ID)
  without a breaking change. That's fine for this scope.
- **`error.validation.failed` left the wire contract.** It is not in the `ErrorCode` enum
  or `Codes()`, but stays as an internal code: it selects 422 in `codeStatus` and is the
  `code` of the `request completed` log record.
- **The response writer picks the shape by code:** `errorBody` in
  `internal/handler/response.go` writes the array for `CodeValidationFailed` and `{code}`
  for everything else. A validation error with no fields is a bug and is answered as a
  500 `error.internal`, so the array never breaks `minItems: 1`.

## 6. Two-operand operations

Every two-operand operation shares the same contract; only the arithmetic differs.

- **Request:** `BinaryOperationRequest`, `{"leftOperand", "rightOperand", "precision"?}`.
- **Responses:** 200 `{"result"}`, 400 invalid JSON, 413, 415, 422 (field-error array or a
  math `{code}`), 500. Same response `$ref`s in the spec for every operation.
- **Validation:** one shared handler, `decodeBinaryRequest` (sections 4 and 5). It is fully
  tested once, in `TestAdd`; each other operation's HTTP test keeps a few rows that prove its
  route uses it.
- **Result check:** the shared service helper `finish` (section 3): no NaN/±Inf, rounding,
  no `-0`.
- **Handler:** one generic `binary(op)` serves every operation; the route is
  `r.POST(PathX, binary(calc.X))`.

### 6.1 Implemented operations

| Operation | Path | Service method | Result | Math errors (422) |
|---|---|---|---|---|
| add | `POST /api/v1/add` | `Calculator.Add` | `leftOperand + rightOperand` | `result.overflow` |
| subtract | `POST /api/v1/subtract` | `Calculator.Subtract` | `leftOperand - rightOperand` | `result.overflow` |
| multiply | `POST /api/v1/multiply` | `Calculator.Multiply` | `leftOperand * rightOperand` | `result.overflow` |

Notes per operation:

- **add:** overflows when both operands are large with the same sign
  (`1.7e308 + 1.7e308`, `-1.7e308 + -1.7e308`). Can't produce NaN from finite operands.
- **subtract:** operand order matters (`3 - 5 = -2`). Overflows when the operands are large
  with opposite signs (`1.7e308 - -1.7e308`, `-1.7e308 - 1.7e308`). A number minus itself
  is `0`, and `-0 - 0` is normalized to `0`. Can't produce NaN from finite operands.
- **multiply:** reaches both float64 limits far more easily than a sum, because exponents
  add up:
  - *Upper:* moderate operands overflow (`1e155 * 1e155`, `-1e155 * 1e155`); the largest
    in-range product of equal powers of ten is `1e154 * 1e154 = 1e308`, and even
    `MaxFloat64 * 1.0000000000000002` overflows. Both signs give 422 `result.overflow`.
  - *Lower:* it is the first operation where two non-zero operands give `0`
    (`1e-200 * 1e-200`, `5e-324 * 0.5`). Accepted per the underflow policy (section 3);
    results down to `5e-324` are kept as subnormals (`1e-160 * 1e-160 = 1e-320`).
    `5e-324 * 1.7976931348623157e308` stays in range (`8.88e-16`).
  - *Precision:* exact integer operands can give an inexact product above 2⁵³
    (`94906267²` → `9007199515875288`, exact `…289`). Accepted.
  - *Sign:* `0 * -5` and `-1e-200 * 1e-200` produce `-0`, normalized to `0`. Can't produce
    NaN from finite operands (`0 * ±Inf` needs an infinite operand, which is rejected).

### 6.2 Adding a two-operand operation

1. **Spec:** add the path to `api/openapi.yaml`, copying an existing operation's request
   body and responses; change `operationId`, `summary` and `description`. List any new
   math error code in the 422 description.
2. **Service:** add `func (*Calculator) X(a, b float64, p Precision) (float64, error)` that
   computes the raw result and passes it through `finish`, wrapping errors with context
   (`fmt.Errorf("x %g and %g: %w", a, b, err)`). Operation-specific failures (e.g.
   division by zero) return an `apperror` sentinel before `finish`.
3. **Handler:** add `X` to the `Calculator` interface, `PathX` to `constants.go`, and
   `r.POST(PathX, binary(calc.X))` to `routes.go`. Add `X` to `fakeCalculator`.
4. **Tests:** a service `TestX` table (results, operand order, rounding, overflow in both
   directions, `-0`, any operation-specific error); a handler `TestX` table (results, math
   errors, a few validation rows); add rows to `TestOperationsNeverReturnNegativeZero` and
   the operation to `TestOperationServiceErrors`.
5. **Docs:** a row in 6.1 and its notes, examples in 6.3, and the "Implemented so far" line
   in `AGENTS.md`.

### 6.3 Examples

Shared by every two-operand operation (`<op>` is `add`, `subtract`, `multiply`, ...):

```
POST /api/v1/<op>
{"leftOperand": 1e309, "rightOperand": "x"}
  → 422 [{"field":"leftOperand","code":"error.field.out.of.range"},
         {"field":"rightOperand","code":"error.field.invalid.type"}]

{"leftOperand": 1, "rightOperand"
  → 400 {"code":"error.request.invalid.json"}

{"leftOperand": 1, "rightOperand": 2, "precision": 2.5}
  → 422 [{"field":"precision","code":"error.field.invalid.type"}]

{"zzz": 0, "precision": -1, "rightOperand": "x"}
  → 422 [{"field":"leftOperand","code":"error.field.required"},
         {"field":"rightOperand","code":"error.field.invalid.type"},
         {"field":"precision","code":"error.field.out.of.range"},
         {"field":"zzz","code":"error.field.unknown"}]
```

`POST /api/v1/add`:

```
{"leftOperand": 1.5, "rightOperand": 2}
  → 200 {"result": 3.5}

{"leftOperand": 0.1, "rightOperand": 0.2}
  → 200 {"result": 0.30000000000000004}

{"leftOperand": 0.1, "rightOperand": 0.2, "precision": 2}
  → 200 {"result": 0.3}

{"leftOperand": 1e-400, "rightOperand": -0}
  → 200 {"result": 0}

{"leftOperand": 1.7e308, "rightOperand": 1.7e308}
  → 422 {"code":"error.calculation.result.overflow"}
```

`POST /api/v1/subtract`:

```
{"leftOperand": 5, "rightOperand": 1.5}
  → 200 {"result": 3.5}

{"leftOperand": 3, "rightOperand": 5}
  → 200 {"result": -2}

{"leftOperand": 0.3, "rightOperand": 0.1}
  → 200 {"result": 0.19999999999999998}

{"leftOperand": 0.3, "rightOperand": 0.1, "precision": 2}
  → 200 {"result": 0.2}

{"leftOperand": -0, "rightOperand": 0}
  → 200 {"result": 0}

{"leftOperand": -1.7e308, "rightOperand": 1.7e308}
  → 422 {"code":"error.calculation.result.overflow"}
```

`POST /api/v1/multiply`:

```
{"leftOperand": 6, "rightOperand": 7}
  → 200 {"result": 42}

{"leftOperand": 1.1, "rightOperand": 1.1}
  → 200 {"result": 1.2100000000000002}

{"leftOperand": 1.1, "rightOperand": 1.1, "precision": 2}
  → 200 {"result": 1.21}

{"leftOperand": 1e154, "rightOperand": 1e154}
  → 200 {"result": 1e+308}

{"leftOperand": 1e155, "rightOperand": -1e155}
  → 422 {"code":"error.calculation.result.overflow"}

{"leftOperand": 1e-160, "rightOperand": 1e-160}
  → 200 {"result": 1e-320}

{"leftOperand": 1e-200, "rightOperand": 1e-200}
  → 200 {"result": 0}

{"leftOperand": 94906267, "rightOperand": 94906267}
  → 200 {"result": 9007199515875288}
```

### 6.4 Implementation notes

`service.Precision`'s zero value means "no rounding"; `service.RoundTo(n)` sets it.

`error.validation.failed` stays as an internal code: it selects the 422 status in
`codeStatus` and is the `code` of the request log record, but it is not in `Codes()` or
the spec and is never sent.

## 7. Open questions

- **`sqrt`:** separate endpoint, designed later. Known so far: it takes a positive integer.
