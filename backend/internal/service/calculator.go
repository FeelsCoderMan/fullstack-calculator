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

// Divide returns a / b, rounded to p. It fails with
// apperror.ErrDivisionByZero when b is 0 or -0, checked before dividing so
// that x/0 is never reported as an overflow (±Inf) or 0/0 as undefined (NaN).
// It fails with apperror.ErrResultOverflow when the quotient is beyond the
// float64 range (1 / 5e-324).
func (*Calculator) Divide(a, b float64, p Precision) (float64, error) {
	if b == 0 { // also matches -0
		return 0, fmt.Errorf("divide %g by %g: %w", a, b, apperror.ErrDivisionByZero)
	}
	r, err := finish(a/b, p)
	if err != nil {
		return 0, fmt.Errorf("divide %g by %g: %w", a, b, err)
	}
	return r, nil
}

// Power returns a raised to the power b, rounded to p. It fails with
// apperror.ErrZeroNegativePower when a is 0 (or -0) and b is negative (that is
// 1/0), checked before computing so it isn't reported as an overflow (±Inf);
// with apperror.ErrResultUndefined for a negative a and a fractional b (no
// real result); and with apperror.ErrResultOverflow beyond the float64 range.
// x^0 is 1 for every x, including 0^0.
func (*Calculator) Power(a, b float64, p Precision) (float64, error) {
	if a == 0 && b < 0 { // also matches a == -0
		return 0, fmt.Errorf("raise %g to %g: %w", a, b, apperror.ErrZeroNegativePower)
	}
	r, err := finish(pow(a, b), p)
	if err != nil {
		return 0, fmt.Errorf("raise %g to %g: %w", a, b, err)
	}
	return r, nil
}

// Sqrt returns the square root of x, rounded to p. It fails with
// apperror.ErrNegativeSquareRoot when x is negative, checked before computing
// so it isn't reported as undefined (math.Sqrt returns NaN). -0 is not
// negative; its root is 0. math.Sqrt is exactly rounded, and a square root
// can't overflow or underflow. The API accepts only whole numbers for x; that
// rule is request validation, enforced by the handler.
func (*Calculator) Sqrt(x float64, p Precision) (float64, error) {
	if x < 0 {
		return 0, fmt.Errorf("square root of %g: %w", x, apperror.ErrNegativeSquareRoot)
	}
	r, err := finish(math.Sqrt(x), p)
	if err != nil {
		return 0, fmt.Errorf("square root of %g: %w", x, err)
	}
	return r, nil
}

// pow returns a^b. math.Pow is not correctly rounded: it is off in the last
// digits for most integer exponents (10^308 is 1.0000000000000006e308) and
// drifts further for bases near 1 (1.0000000001^1e10 is wrong from the 8th
// digit). So every integer exponent below maxExactExponent in magnitude goes
// through intPow. A fractional exponent, an integer one at or beyond
// maxExactExponent (where every base other than 0 and ±1 overflows or
// underflows), and a NaN base (which big.Float can't hold) use math.Pow.
func pow(a, b float64) float64 {
	if math.IsNaN(a) || b != math.Trunc(b) || math.Abs(b) >= maxExactExponent {
		return math.Pow(a, b)
	}
	return intPow(a, uint64(math.Abs(b)), b < 0)
}

// intPow returns a^n, or a^-n when negative, by exponentiation by squaring in
// a powPrecision-bit big.Float, rounded once to float64. The working precision
// is so far beyond float64's 53 bits that the result is the nearest float64 to
// the true power (barring a tie within ~2^-2000, which never occurs in
// practice). big.Float turns exponent overflow into ±Inf and underflow into
// ±0, and Float64 rounds subnormals correctly. a must be finite, and not 0
// when negative (the caller rejects 0^negative).
func intPow(a float64, n uint64, negative bool) float64 {
	base := new(big.Float).SetPrec(powPrecision).SetFloat64(a)
	r := new(big.Float).SetPrec(powPrecision).SetInt64(1)
	for n > 0 {
		if n&1 == 1 {
			r.Mul(r, base)
		}
		n >>= 1
		if n > 0 {
			base.Mul(base, base)
		}
	}
	if negative {
		r.Quo(new(big.Float).SetPrec(powPrecision).SetInt64(1), r)
	}
	f, _ := r.Float64() // the accuracy (above or below) is irrelevant: f is the nearest float64
	return f
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
