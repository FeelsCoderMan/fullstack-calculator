// Package service holds the calculator's business logic: float64 arithmetic,
// result checks and rounding. It knows nothing about HTTP or JSON; failures are
// returned as apperror sentinels wrapped with context.
package service

import (
	"fmt"
	"math"
	"math/big"
	"strconv"

	"fullstack-calculator/backend/internal/apperror"
)

// Precision is the number of decimal places to round a result to. The zero
// value means no rounding (the full float64 result); use RoundTo to round.
type Precision struct {
	places int
	set    bool
}

// RoundTo returns a Precision that rounds to places decimal places. Callers
// validate places against MinPrecision and MaxPrecision first.
func RoundTo(places int) Precision {
	return Precision{places: places, set: true}
}

// Calculator performs the arithmetic operations. It is stateless; the zero
// value is ready to use.
type Calculator struct{}

// NewCalculator returns a Calculator.
func NewCalculator() *Calculator {
	return &Calculator{}
}

// Add returns a + b, rounded to p. It fails with apperror.ErrResultOverflow
// when the sum is beyond the float64 range.
func (*Calculator) Add(a, b float64, p Precision) (float64, error) {
	r, err := finish(a+b, p)
	if err != nil {
		return 0, fmt.Errorf("add %g and %g: %w", a, b, err)
	}
	return r, nil
}

// Subtract returns a - b, rounded to p. It fails with
// apperror.ErrResultOverflow when the difference is beyond the float64 range.
func (*Calculator) Subtract(a, b float64, p Precision) (float64, error) {
	r, err := finish(a-b, p)
	if err != nil {
		return 0, fmt.Errorf("subtract %g from %g: %w", b, a, err)
	}
	return r, nil
}

// Multiply returns a * b, rounded to p. It fails with
// apperror.ErrResultOverflow when the product is beyond the float64 range. A
// product below about 2.5e-324, half the smallest positive float64
// (1e-200 * 1e-200), underflows to 0, which is accepted (api/openapi.yaml).
func (*Calculator) Multiply(a, b float64, p Precision) (float64, error) {
	r, err := finish(a*b, p)
	if err != nil {
		return 0, fmt.Errorf("multiply %g by %g: %w", a, b, err)
	}
	return r, nil
}

// finish turns a raw float64 result into an API result. It runs on every
// operation's result, in this order:
//  1. NaN and ±Inf are errors, never results.
//  2. Round to p, if set.
//  3. -0 becomes 0, last, so a rounding that yields -0 (-0.001 at precision 2)
//     is caught too.
func finish(r float64, p Precision) (float64, error) {
	switch {
	case math.IsNaN(r):
		return 0, apperror.ErrResultUndefined
	case math.IsInf(r, 0):
		return 0, apperror.ErrResultOverflow
	}
	if p.set {
		var err error
		if r, err = round(r, p.places); err != nil {
			return 0, err
		}
	}
	if r == 0 {
		return 0, nil // also matches -0, which this replaces with +0
	}
	return r, nil
}

// round rounds a finite r to places decimal places, half away from zero, as a
// person would on paper. It rounds r's shortest decimal form (the digits
// strconv prints, e.g. 2.675), not its exact binary value (2.67499999...), so
// 2.675 rounds to 2.68 and 2.5 to 3. The work is done on an exact rational, so
// it can't overflow, unlike math.Round(r*10^p)/10^p.
func round(r float64, places int) (float64, error) {
	shortest := strconv.FormatFloat(r, shortestFormat, shortestPrecision, bitSize)
	exact, ok := new(big.Rat).SetString(shortest)
	if !ok {
		return 0, apperror.Internal(fmt.Errorf("round %g: parse %q: %w", r, shortest, errNotRational))
	}
	// FloatString rounds the last digit half away from zero.
	rounded, err := strconv.ParseFloat(exact.FloatString(places), bitSize)
	if err != nil {
		return 0, apperror.Internal(fmt.Errorf("round %g to %d places: %w", r, places, err))
	}
	return rounded, nil
}
