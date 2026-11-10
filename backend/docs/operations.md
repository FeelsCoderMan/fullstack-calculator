# Operation endpoints: design notes

Status: **implemented for every operation: `add`, `subtract`, `multiply`, `divide`, `power`,
`percentage` and `sqrt`.** The normative rules are in `AGENTS.md` and
`api/openapi.yaml`; this file keeps the decisions and their rationale for each
operation.

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

`sqrt` takes one operand instead: `{"operand": 16, "precision": 2}` (section 7).

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
| Result NaN, e.g. `(-8) ^ 0.5` | `math.IsNaN(result)` | 422 `error.calculation.result.undefined` |
| Division by zero: divisor `0`, `-0`, or one that underflows to `0` (`1e-400`) | `b == 0` in `Divide`, **before** dividing | 422 `error.calculation.division.by.zero` (never reported as overflow or undefined) |
| Square root of a negative number, e.g. `sqrt(-4)` | `x < 0` in `Sqrt`, **before** computing | 422 `error.calculation.negative.square.root` (never reported as undefined) |
| Zero to a negative power: base `0`/`-0` (or one that underflows to `0`), exponent `< 0` | `a == 0 && b < 0` in `Power`, **before** computing | 422 `error.calculation.zero.negative.power` (never reported as overflow) |
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
| divide | `POST /api/v1/divide` | `Calculator.Divide` | `leftOperand / rightOperand` | `division.by.zero`, `result.overflow` |
| power | `POST /api/v1/power` | `Calculator.Power` | `leftOperand ^ rightOperand` | `zero.negative.power`, `result.undefined`, `result.overflow` |
| percentage | `POST /api/v1/percentage` | `Calculator.Percentage` | `leftOperand` percent of `rightOperand` = `L * R / 100` | `result.overflow` |

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
- **divide:** the first operation with its own error. Operand order matters (`4 / 10 = 0.4`).
  - *Division by zero:* `rightOperand` equal to `0` returns 422 `division.by.zero`. It is
    checked **before** dividing, because IEEE division would otherwise give `1/0 = +Inf`,
    `-1/0 = 1/-0 = -Inf` (reported wrongly as overflow) and `0/0 = NaN` (reported wrongly
    as undefined). `-0` matches (`-0 == 0`). A divisor that underflows on input
    (`1e-400` → `0`) is division by zero too, consistent with the underflow policy. An
    invalid divisor (`"0"`, `null`) is a field error, not division by zero. The check runs
    before rounding, so `precision` never hides it.
  - *Upper:* a tiny non-zero divisor overflows (`1 / 5e-324`, `1e300 / 1e-10`,
    `MaxFloat64 / 0.5`, either sign): 422 `result.overflow`. `1 / 1e-308 = 1e308` stays in
    range.
  - *Lower:* a huge divisor underflows to `0` (`1e-300 / 1e300`, `5e-324 / 2`,
    `5e-324 / MaxFloat64`), accepted; `1e-310 / 10 = 1e-311` is kept as a subnormal.
  - *Precision:* repeating decimals are cut at float64 precision (`1 / 3 =
    0.3333333333333333`); float noise shows up (`0.3 / 0.1 = 2.9999999999999996`), and
    `precision` fixes it (`3` at precision 2).
  - *Sign:* `0 / -5` and `-1e-300 / 1e300` produce `-0`, normalized to `0`. With the zero
    check, it can't produce NaN from finite operands.
- **power:** `leftOperand` is the base, `rightOperand` the exponent.
  - *Zero to a negative power* (`0 ^ -1`, `-0 ^ -2`, `0 ^ -0.5`, and `1e-400 ^ -1` whose
    base underflows to `0`) is `1 / 0`: its own code, 422 `zero.negative.power` (chosen over
    reusing `division.by.zero`, because the user didn't divide). Checked before computing,
    because `math.Pow` returns ±Inf, which would be reported as overflow.
  - *`x ^ 0 = 1`* for every `x`, including `0 ^ 0` and `-0 ^ 0` (IEEE 754, JavaScript's
    `Math.pow`). `1 ^ y = 1` and `(-1) ^ (huge even y) = 1`.
  - *No real result:* a negative base with a fractional exponent is NaN, 422
    `result.undefined` (`(-8) ^ 0.5`, `(-2) ^ 1.5`). `(-8) ^ (1/3)` is undefined too: `1/3`
    isn't exact in float64, so the exponent is fractional, not a cube root.
  - *Exactness:* `math.Pow` is not correctly rounded: it gets **505 of the 632** float64
    powers of ten wrong (`10 ^ 308` = `1.0000000000000006e308`), and for bases near 1 it
    drifts much further (`1.0000000001 ^ 1e10`: `2.718282043475248` instead of
    `2.7182820532347876`, wrong from the 8th digit). So every integer exponent with
    `|b| < 2^64` uses exponentiation by squaring in a 2048-bit `big.Float`, rounded once to
    the nearest float64 (`intPow`, at most 63 squarings, ~90µs worst case);
    `TestPowerOfTenIsExact` checks all 632 powers of ten. At and beyond `2^64` every base
    other than `0` and `±1` overflows or underflows (`(1+2^-52) ^ 2^64` ≈ `e^4096`), so
    `math.Pow` is exact there. Fractional exponents use `math.Pow` and can be off in the
    last digit. `big.Float` turns exponent overflow into ±Inf and underflow into ±0, so
    the limits behave the same on both paths. A first version used `2^30` as the limit;
    review found the near-1 drift, hence `2^64`.
  - *Upper:* `2 ^ 1023 = 8.98846567431158e307` is in range; `2 ^ 1024`, `10 ^ 309`,
    `(-10) ^ 309` (negative), `MaxFloat64 ^ 2`, `5e-324 ^ -1`, `1e-300 ^ -2` and
    `1.0000001 ^ 1e10` overflow: 422 `result.overflow`.
  - *Lower:* `2 ^ -1074 = 5e-324` and `10 ^ -323 = 1e-323` are kept as subnormals;
    `2 ^ -1075`, `10 ^ -400` and `0.5 ^ 1e10` underflow to `0`, accepted.
  - *Sign:* `-0 ^ 3` and `(-10) ^ -401` produce `-0`, normalized to `0`.
- **percentage:** `leftOperand` percent of `rightOperand`, `L * R / 100` (the definition in
  AGENTS.md). Commutative: `50% of 10 = 10% of 50 = 5`. Negative percentages and bases are
  allowed (`-20% of 50 = -10`).
  - *Exactness:* no float64 order of operations is right. `(L * R) / 100` overflows when
    only the product is too large (`1e307% of 50` → `+Inf`, true `5e306`;
    `100% of MaxFloat64` → `+Inf`). `L / 100 * R` underflows or adds noise
    (`5e-324% of 100` → `0`, true `5e-324`; `7% of 100` → `7.000000000000001`). On
    100,000 random two-decimal inputs, **each** order is off in the last digit about a
    quarter of the time (`(L * R) / 100` 26%, `L / 100 * R` 25–26%; both at once ~9%, at
    least one ~43%), e.g. `15% of 19.99` → `2.9984999999999995` in both, true `2.9985`. So
    `percentOf` computes `L * R / 100` exactly as a `big.Rat` and rounds once to the
    nearest float64 (`Rat.Float64`, which returns ±Inf beyond the range). Binary inputs
    that aren't exact still show (`0.1% of 0.3` = `0.00030000000000000003`, the true value
    of the binary `0.1` and `0.3`; here both float64 orders happen to print `0.0003`, by
    a lucky rounding); `precision` fixes it. A non-finite operand (not reachable through the API) falls back to float64
    math, because `big.Rat` can't hold it.
  - *Upper:* only a true result beyond the range overflows: `200% of MaxFloat64`,
    `-1e308% of 1e308` → 422 `result.overflow`.
  - *Lower:* `5e-324% of 100 = 5e-324` is kept; `5e-324% of 50` (exactly half the
    smallest float, a tie) and `1e-300% of 1e-300` underflow to `0`, accepted.
  - *Sign:* `0% of -5` and `-1e-300% of 1e-300` give `0`, never `-0`.

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

Shared by every two-operand operation (`<op>` is `add`, `subtract`, `multiply`, `divide`, `power`, `percentage`):

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

`POST /api/v1/divide`:

```
{"leftOperand": 10, "rightOperand": 4}
  → 200 {"result": 2.5}

{"leftOperand": 1, "rightOperand": 3}
  → 200 {"result": 0.3333333333333333}

{"leftOperand": 2, "rightOperand": 3, "precision": 2}
  → 200 {"result": 0.67}

{"leftOperand": 1, "rightOperand": 0}
  → 422 {"code":"error.calculation.division.by.zero"}

{"leftOperand": 0, "rightOperand": 0}
  → 422 {"code":"error.calculation.division.by.zero"}

{"leftOperand": 1, "rightOperand": 1e-400}
  → 422 {"code":"error.calculation.division.by.zero"}

{"leftOperand": 1, "rightOperand": 5e-324}
  → 422 {"code":"error.calculation.result.overflow"}

{"leftOperand": 1e-300, "rightOperand": 1e300}
  → 200 {"result": 0}
```

`POST /api/v1/power`:

```
{"leftOperand": 2, "rightOperand": 10}
  → 200 {"result": 1024}

{"leftOperand": 2, "rightOperand": 0.5, "precision": 3}
  → 200 {"result": 1.414}

{"leftOperand": 10, "rightOperand": 308}
  → 200 {"result": 1e+308}

{"leftOperand": 0, "rightOperand": 0}
  → 200 {"result": 1}

{"leftOperand": 0, "rightOperand": -1}
  → 422 {"code":"error.calculation.zero.negative.power"}

{"leftOperand": -8, "rightOperand": 0.5}
  → 422 {"code":"error.calculation.result.undefined"}

{"leftOperand": 10, "rightOperand": 309}
  → 422 {"code":"error.calculation.result.overflow"}

{"leftOperand": 10, "rightOperand": -400}
  → 200 {"result": 0}
```

`POST /api/v1/percentage`:

```
{"leftOperand": 10, "rightOperand": 50}
  → 200 {"result": 5}

{"leftOperand": 15, "rightOperand": 19.99}
  → 200 {"result": 2.9985}

{"leftOperand": 7, "rightOperand": 100}
  → 200 {"result": 7}

{"leftOperand": -20, "rightOperand": 50}
  → 200 {"result": -10}

{"leftOperand": 1e307, "rightOperand": 50}
  → 200 {"result": 5e+306}

{"leftOperand": 200, "rightOperand": 1.7976931348623157e308}
  → 422 {"code":"error.calculation.result.overflow"}

{"leftOperand": 5e-324, "rightOperand": 100}
  → 200 {"result": 5e-324}
```

### 6.4 Implementation notes

`service.Precision`'s zero value means "no rounding"; `service.RoundTo(n)` sets it.

`error.validation.failed` stays as an internal code: it selects the 422 status in
`codeStatus` and is the `code` of the request log record, but it is not in `Codes()` or
the spec and is never sent.

## 7. One-operand operations: `sqrt`

| Operation | Path | Service method | Result | Math errors (422) |
|---|---|---|---|---|
| sqrt | `POST /api/v1/sqrt` | `Calculator.Sqrt` | `√operand` | `negative.square.root` |

Decisions (made with the project owner):

- **Request:** `UnaryOperationRequest`, `{"operand", "precision"?}`. `operand` was chosen
  over `radicand` and `value`. `precision` works as for every other operation.
- **Domain: whole numbers.** `operand` must be an integer; `2.0` and `1e20` count, `2.25`
  is the field code `error.field.invalid.type` (as for a non-integer `precision`). `0` is
  allowed. The rule is request validation, enforced by the handler (`fieldReader.integer`);
  `Calculator.Sqrt` itself accepts any float64.
- **Negative operand:** a math error, not a field error: 422
  `error.calculation.negative.square.root` (the code already in the catalog). Validation
  runs first, so a negative decimal (`-2.5`) is `invalid.type`, while `-4` is
  `negative.square.root`. Checked before computing, because `math.Sqrt(-4)` is NaN, which
  would be reported as undefined. `-0` is not negative: its root is `0`.
- **Validation:** its own decoder, `decodeUnaryRequest`, reports every problem at once in
  the order `operand`, `precision`, then unknown properties alphabetically. Two-operand
  fields (`leftOperand`) are unknown here. The whole-number rule runs after parsing, so an
  operand that underflows to `0` (`1e-400`) passes and gives `0`, while a tiny decimal that
  float64 can hold (`1e-320`) is `invalid.type`.
- **Adding another one-operand operation:** follow 6.2, with these differences: the spec
  uses `UnaryOperationRequest`; the service method is `X(x float64, p Precision)`; the
  route is `r.POST(PathX, unary(calc.X))`; the `operations` row in
  `TestOperationServiceErrors` needs a one-operand `body`; and if the operand isn't a
  whole number, add a field reader for its rule.

Limits:

- *Upper:* none. `math.Sqrt` is exactly rounded (IEEE 754), and the root of
  `MaxFloat64` is `1.3407807929942596e154`, so it can't overflow.
- *Lower:* none. The root of a whole number ≥ 1 is ≥ 1; `0` gives `0`. An operand that
  underflows on input (`1e-400`) becomes `0` and gives `0`.
- *Precision:* integers above 2⁵³ are read inexactly (`9007199254740993` → `…992`), as
  everywhere. `94906267²` is exactly `…289`; float64 holds it as `…288`, whose root rounds
  to `94906267`.
- *Sign:* `sqrt(-0)` is `-0`, normalized to `0`.

Examples, `POST /api/v1/sqrt`:

```
{"operand": 16}
  → 200 {"result": 4}

{"operand": 2}
  → 200 {"result": 1.4142135623730951}

{"operand": 2, "precision": 3}
  → 200 {"result": 1.414}

{"operand": 1.7976931348623157e308}
  → 200 {"result": 1.3407807929942596e+154}

{"operand": -4}
  → 422 {"code":"error.calculation.negative.square.root"}

{"operand": 2.25}
  → 422 [{"field":"operand","code":"error.field.invalid.type"}]

{"zeta": 1, "precision": 16, "operand": 2.5, "alpha": 1}
  → 422 [{"field":"operand","code":"error.field.invalid.type"},
         {"field":"precision","code":"error.field.out.of.range"},
         {"field":"alpha","code":"error.field.unknown"},
         {"field":"zeta","code":"error.field.unknown"}]
```

## 8. Open questions

None for the operations. Every endpoint in the AGENTS.md table is implemented.
