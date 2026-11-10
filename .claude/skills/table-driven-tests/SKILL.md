---
name: table-driven-tests
description: Project conventions for writing table-driven unit tests in Go (slice of named cases with inputs and expectations side by side, t.Run subtests, got/want messages, go-cmp diffs) and Vitest (it.each with named rows). Load before writing or modifying any test in this repo.
---

# Table-driven tests

The single source of truth for how tests are written in this repo. Based on Dave Cheney,
[*Prefer table driven tests*](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests).
Inputs, expected outputs, and a name live in a **table**, and one harness runs every row.
Adding a case means adding a row, not copy-pasting a test function.

**The goal is readability: a reader sees what a case does and what it expects from the
case itself**, without reading the harness.

## Go

### Canonical example

This is the reference style; copy its shape. It comes from
`backend/internal/handler/middleware_test.go`:

```go
func TestAccessLogRecordsRequest(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus float64
	}{
		{
			name:       "explicit status",
			handler:    func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) },
			wantStatus: http.StatusTeapot,
		},
		{
			name:       "implicit 200 on write",
			handler:    func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("hi")) },
			wantStatus: http.StatusOK,
		},
		{
			name: "1xx is not the final status",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusEarlyHints)
				w.WriteHeader(http.StatusCreated)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "only first WriteHeader counts",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusAccepted)
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatus: http.StatusAccepted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange → act → assert, using only tt.* fields
		})
	}
}
```

What makes it good:
- **Each case is a self-contained block**: `name`, then the inputs, then the `want*` fields.
  The expectation sits right under the input that produces it.
- **Every field is keyed** (`name:`, `handler:`, `wantStatus:`), never positional.
- **The name states the behaviour being pinned down** ("only first WriteHeader counts"),
  so a failure tells you which rule broke.
- **Inputs are concrete and visible** in the case (even small inline funcs), not hidden
  in setup code far away.

### Rules

1. **The table is a slice of anonymous structs with a `name` field**:
   `tests := []struct{ name string; ...; want... }{...}`, iterated with
   `for _, tt := range tests` and `t.Run(tt.name, ...)`. Cases run and are reported in
   the order they're written, which reads like a spec.
2. **One case per block, all fields keyed, expectations last.** Field order is
   `name` → inputs → `want*`. Expected fields are prefixed `want` (`want`, `wantErr`,
   `wantStatus`, `wantBody`). Omit zero-value fields only when "zero" is the obvious
   meaning (e.g. no `wantErr` means success).
3. **Names describe the scenario**, lowercase with spaces (`"division by zero"`,
   `"missing b and a not a number"`), never `"test1"` or `"case 3"`. Names are unique and
   selectable: `go test -run 'TestDivide/division_by_zero' -v`.
4. **Every case runs in `t.Run(tt.name, ...)`.** A failing case never hides the others.
   Inside a subtest, use `t.Fatalf` when continuing makes no sense (e.g. an error happened,
   so the result is meaningless) and `t.Errorf` for independent assertions on one result.
5. **Failure messages follow `Func(inputs) = got, want want`.** Input first, then got,
   then want. Use `%v` for scalars and `%#v` when type or quoting matters (strings, slices).
   Don't print pointers with `%v`; dereference or diff them.
6. **Compare composite values with `go-cmp`**, not `reflect.DeepEqual`:
   ```go
   if diff := cmp.Diff(tt.want, got); diff != "" {
       t.Errorf("Validate(%s) mismatch (-want +got):\n%s", tt.body, diff)
   }
   ```
   `github.com/google/go-cmp` is an approved **test-only** dependency. Add it with
   `go get github.com/google/go-cmp/cmp` the first time it's needed. Plain `==` is fine for
   scalars and comparable structs.
7. **Errors are matched by identity**, never by message text: `errors.Is(err, tt.wantErr)`
   against sentinels from the package's `errors.go`. A `nil` `wantErr` means success is
   expected. Use a `wantErr bool` only when no sentinel exists.
8. **One table per behaviour under test.** Don't merge unrelated behaviours into one giant
   table with flag fields that switch the harness logic (`if tt.isSpecial {...}`). If the
   harness needs branches, split the table.
9. **Shared setup goes above the loop or into a helper** marked `t.Helper()`. Helpers that
   build fixtures belong in the `_test.go` file of the package that uses them.
10. **Use coverage to find missing rows, then think about boundaries.** Run
    `go test ./... -cover`, look at uncovered branches, and add rows for the edges:
    zero, negative, very large/small (`math.MaxFloat64`, `math.SmallestNonzeroFloat64`),
    NaN/±Inf producing inputs, empty/missing/null/wrong-type fields, and unknown fields.
11. **Not everything is a table.** Lifecycle, concurrency, and timing tests (graceful
    shutdown, draining, race checks) are clearer as straight-line tests. Don't force them
    into a table with one row.

### HTTP handler tables

Handler tests assert the **exact wire contract**: status code and full JSON body, written
as **literal JSON strings** in each case. This is the allowed exception to the "no
hardcoded strings" rule (AGENTS.md): the literal catches an accidental change to a constant.

```go
tests := []struct {
	name       string
	method     string
	path       string
	body       string
	wantStatus int
	wantBody   string
}{
	{
		name:       "adds two numbers",
		method:     http.MethodPost,
		path:       "/api/v1/add",
		body:       `{"leftOperand":1,"rightOperand":2}`,
		wantStatus: http.StatusOK,
		wantBody:   `{"result":3}`,
	},
	{
		name:       "missing both operands",
		method:     http.MethodPost,
		path:       "/api/v1/add",
		body:       `{}`,
		wantStatus: http.StatusUnprocessableEntity,
		wantBody:   `[{"field":"leftOperand","code":"error.field.required"},{"field":"rightOperand","code":"error.field.required"}]`,
	},
}
```

Compare JSON semantically (unmarshal both sides and `cmp.Diff`) so whitespace and key
order don't cause false failures. Keep the order of the field-error array significant, because the API
guarantees a stable order.

## TypeScript (Vitest)

Same shape: an array of keyed rows with `name` first and expectations last, passed to
`it.each` with `$name` as the title.

```ts
const cases = [
  {
    name: 'empty input',
    input: '',
    want: 'error.field.required',
  },
  {
    name: 'not a number',
    input: '1.2.3',
    want: 'error.field.invalid.type',
  },
  {
    name: 'valid negative decimal',
    input: '-4.5',
    want: null,
  },
] as const;

describe('validateOperand', () => {
  it.each(cases)('$name', ({ input, want }) => {
    expect(validateOperand(input)).toBe(want);
  });
});
```

- Use `toEqual` for objects and arrays (Vitest prints a diff), and `toBe` for primitives.
- Component tests may use tables for input/output pairs (typed value → rendered message).
  User-flow tests (type, click, wait) stay straight-line.
- Mock the API client in component tests. Test the API client itself with a mocked `fetch`
  table (status + body → returned value or thrown error).

## Checklist before finishing

- [ ] Every new or changed behaviour has rows for success **and** each error branch
- [ ] Each case is a keyed block: `name` → inputs → `want*`; names are unique sentences
- [ ] Failure messages show inputs, got, and want
- [ ] Errors matched with `errors.Is` (Go), composite values with `cmp.Diff` / `toEqual`
- [ ] `go test ./... -race -cover` (or `npm test`) run, and the real output reported
