package service

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"testing"

	"fullstack-calculator/backend/internal/apperror"
)

func TestAdd(t *testing.T) {
	negZero := math.Copysign(0, -1)

	tests := []struct {
		name    string
		a, b    float64
		p       Precision
		want    float64
		wantErr error
	}{
		{
			name: "integers",
			a:    1,
			b:    2,
			want: 3,
		},
		{
			name: "decimals",
			a:    1.5,
			b:    2.25,
			want: 3.75,
		},
		{
			name: "negative and positive",
			a:    -7.5,
			b:    2.5,
			want: -5,
		},
		{
			name: "float noise is kept without precision",
			a:    0.1,
			b:    0.2,
			want: 0.30000000000000004,
		},
		{
			name: "opposites sum to positive zero",
			a:    -4,
			b:    4,
			want: 0,
		},
		{
			name: "negative zeros sum to positive zero",
			a:    negZero,
			b:    negZero,
			want: 0,
		},
		{
			name: "smallest subnormals cancel to positive zero",
			a:    -math.SmallestNonzeroFloat64,
			b:    math.SmallestNonzeroFloat64,
			want: 0,
		},
		{
			name: "integer above 2^53 loses precision without error",
			a:    9007199254740992,
			b:    1,
			want: 9007199254740992,
		},
		{
			name: "largest float plus zero",
			a:    math.MaxFloat64,
			b:    0,
			want: math.MaxFloat64,
		},
		{
			name:    "positive overflow",
			a:       math.MaxFloat64,
			b:       math.MaxFloat64,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "negative overflow",
			a:       -math.MaxFloat64,
			b:       -math.MaxFloat64,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name: "rounds to precision",
			a:    0.1,
			b:    0.2,
			p:    RoundTo(2),
			want: 0.3,
		},
		{
			name: "precision zero rounds to an integer",
			a:    1.25,
			b:    1.25,
			p:    RoundTo(0),
			want: 3,
		},
		{
			name: "rounding to negative zero gives positive zero",
			a:    -0.001,
			b:    0,
			p:    RoundTo(2),
			want: 0,
		},
		{
			name: "rounds the decimal the user sees, not the binary value below it",
			a:    2.675,
			b:    0,
			p:    RoundTo(2),
			want: 2.68,
		},
		{
			name: "1.005 rounds up like on paper",
			a:    1,
			b:    0.005,
			p:    RoundTo(2),
			want: 1.01,
		},
		{
			name: "maximum precision keeps a short decimal",
			a:    1.5,
			b:    0.25,
			p:    RoundTo(MaxPrecision),
			want: 1.75,
		},
		{
			name: "rounding the largest float does not overflow",
			a:    math.MaxFloat64,
			b:    0,
			p:    RoundTo(MaxPrecision),
			want: math.MaxFloat64,
		},
		{
			name:    "overflow is reported before rounding",
			a:       math.MaxFloat64,
			b:       math.MaxFloat64,
			p:       RoundTo(2),
			wantErr: apperror.ErrResultOverflow,
		},
	}

	calc := NewCalculator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calc.Add(tt.a, tt.b, tt.p)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Add(%g, %g, %+v) error = %v, want %v", tt.a, tt.b, tt.p, err, tt.wantErr)
			}
			if got != tt.want || math.Signbit(got) != math.Signbit(tt.want) {
				t.Errorf("Add(%g, %g, %+v) = %g, want %g", tt.a, tt.b, tt.p, got, tt.want)
			}
		})
	}
}

func TestSubtract(t *testing.T) {
	negZero := math.Copysign(0, -1)

	tests := []struct {
		name    string
		a, b    float64
		p       Precision
		want    float64
		wantErr error
	}{
		{
			name: "integers",
			a:    5,
			b:    3,
			want: 2,
		},
		{
			name: "operand order matters",
			a:    3,
			b:    5,
			want: -2,
		},
		{
			name: "subtracting a negative adds",
			a:    -1.5,
			b:    -2.25,
			want: 0.75,
		},
		{
			name: "float noise is kept without precision",
			a:    0.3,
			b:    0.1,
			want: 0.19999999999999998,
		},
		{
			name: "rounds to precision",
			a:    0.3,
			b:    0.1,
			p:    RoundTo(2),
			want: 0.2,
		},
		{
			name: "number minus itself is positive zero",
			a:    -2.5,
			b:    -2.5,
			want: 0,
		},
		{
			name: "negative zero minus zero is positive zero",
			a:    negZero,
			b:    0,
			want: 0,
		},
		{
			name: "rounding to negative zero gives positive zero",
			a:    0,
			b:    0.001,
			p:    RoundTo(2),
			want: 0,
		},
		{
			name: "largest float minus zero",
			a:    math.MaxFloat64,
			b:    0,
			want: math.MaxFloat64,
		},
		{
			name: "largest float minus itself",
			a:    math.MaxFloat64,
			b:    math.MaxFloat64,
			want: 0,
		},
		{
			name:    "positive overflow",
			a:       math.MaxFloat64,
			b:       -math.MaxFloat64,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "negative overflow",
			a:       -math.MaxFloat64,
			b:       math.MaxFloat64,
			wantErr: apperror.ErrResultOverflow,
		},
	}

	calc := NewCalculator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calc.Subtract(tt.a, tt.b, tt.p)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Subtract(%g, %g, %+v) error = %v, want %v", tt.a, tt.b, tt.p, err, tt.wantErr)
			}
			if got != tt.want || math.Signbit(got) != math.Signbit(tt.want) {
				t.Errorf("Subtract(%g, %g, %+v) = %g, want %g", tt.a, tt.b, tt.p, got, tt.want)
			}
		})
	}
}

func TestMultiply(t *testing.T) {
	tests := []struct {
		name    string
		a, b    float64
		p       Precision
		want    float64
		wantErr error
	}{
		{
			name: "integers",
			a:    6,
			b:    7,
			want: 42,
		},
		{
			name: "negative times positive is negative",
			a:    -2.5,
			b:    4,
			want: -10,
		},
		{
			name: "two negatives make a positive",
			a:    -3,
			b:    -4,
			want: 12,
		},
		{
			name: "float noise is kept without precision",
			a:    1.1,
			b:    1.1,
			want: 1.2100000000000002,
		},
		{
			name: "rounds to precision",
			a:    1.1,
			b:    1.1,
			p:    RoundTo(2),
			want: 1.21,
		},
		{
			name: "times zero is zero",
			a:    123.45,
			b:    0,
			want: 0,
		},
		{
			name: "zero times a negative is positive zero",
			a:    0,
			b:    -5,
			want: 0,
		},
		{
			name: "rounding to negative zero gives positive zero",
			a:    -0.01,
			b:    0.1,
			p:    RoundTo(2),
			want: 0,
		},
		{
			name: "integers above 2^53 lose precision without error",
			a:    94906267,
			b:    94906267,
			want: 9007199515875288, // exact product is 9007199515875289
		},

		// Upper limit: ±1.7976931348623157e308.
		{
			name: "largest float times one",
			a:    math.MaxFloat64,
			b:    1,
			want: math.MaxFloat64,
		},
		{
			name: "largest float times minus one is the most negative float",
			a:    math.MaxFloat64,
			b:    -1,
			want: -math.MaxFloat64,
		},
		{
			name: "largest product of equal powers of ten",
			a:    1e154,
			b:    1e154,
			want: 1e308,
		},
		{
			name: "smallest float times largest float stays in range",
			a:    math.SmallestNonzeroFloat64,
			b:    math.MaxFloat64,
			want: 8.881784197001251e-16,
		},
		{
			name:    "largest float times the next float above one overflows",
			a:       math.MaxFloat64,
			b:       1.0000000000000002,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "positive overflow from moderate operands",
			a:       1e155,
			b:       1e155,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "negative overflow from moderate operands",
			a:       -1e155,
			b:       1e155,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "largest times most negative overflows negatively",
			a:       math.MaxFloat64,
			b:       -math.MaxFloat64,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "overflow is reported before rounding",
			a:       1e155,
			b:       1e155,
			p:       RoundTo(2),
			wantErr: apperror.ErrResultOverflow,
		},

		// Lower limit: below ~2.5e-324 (half of 5e-324) a product underflows to 0 (accepted).
		{
			name: "subnormal product is kept",
			a:    1e-160,
			b:    1e-160,
			want: 1e-320,
		},
		{
			name: "smallest float times 0.6 rounds up to the smallest float",
			a:    math.SmallestNonzeroFloat64,
			b:    0.6,
			want: math.SmallestNonzeroFloat64,
		},
		{
			name: "smallest float halved underflows to zero",
			a:    math.SmallestNonzeroFloat64,
			b:    0.5,
			want: 0,
		},
		{
			name: "two small non-zero operands underflow to zero",
			a:    1e-200,
			b:    1e-200,
			want: 0,
		},
		{
			name: "negative underflow gives positive zero",
			a:    -1e-200,
			b:    1e-200,
			want: 0,
		},
	}

	calc := NewCalculator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calc.Multiply(tt.a, tt.b, tt.p)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Multiply(%g, %g, %+v) error = %v, want %v", tt.a, tt.b, tt.p, err, tt.wantErr)
			}
			if got != tt.want || math.Signbit(got) != math.Signbit(tt.want) {
				t.Errorf("Multiply(%g, %g, %+v) = %g, want %g", tt.a, tt.b, tt.p, got, tt.want)
			}
		})
	}
}

func TestDivide(t *testing.T) {
	negZero := math.Copysign(0, -1)

	tests := []struct {
		name    string
		a, b    float64
		p       Precision
		want    float64
		wantErr error
	}{
		{
			name: "exact quotient",
			a:    10,
			b:    4,
			want: 2.5,
		},
		{
			name: "operand order matters",
			a:    4,
			b:    10,
			want: 0.4,
		},
		{
			name: "negative divided by positive is negative",
			a:    -7,
			b:    2,
			want: -3.5,
		},
		{
			name: "two negatives make a positive",
			a:    -9,
			b:    -3,
			want: 3,
		},
		{
			name: "repeating decimal is cut at float64 precision",
			a:    1,
			b:    3,
			want: 0.3333333333333333,
		},
		{
			name: "rounds a repeating decimal half away from zero",
			a:    2,
			b:    3,
			p:    RoundTo(2),
			want: 0.67,
		},
		{
			name: "float noise is kept without precision",
			a:    0.3,
			b:    0.1,
			want: 2.9999999999999996,
		},
		{
			name: "rounding removes float noise",
			a:    0.3,
			b:    0.1,
			p:    RoundTo(2),
			want: 3,
		},
		{
			name: "zero divided by a number is zero",
			a:    0,
			b:    7,
			want: 0,
		},
		{
			name: "zero divided by a negative is positive zero",
			a:    0,
			b:    -5,
			want: 0,
		},
		{
			name: "rounding to negative zero gives positive zero",
			a:    -1,
			b:    1000,
			p:    RoundTo(2),
			want: 0,
		},

		// Division by zero: checked before dividing, whatever the dividend.
		{
			name:    "positive by zero",
			a:       1,
			b:       0,
			wantErr: apperror.ErrDivisionByZero,
		},
		{
			name:    "negative by zero",
			a:       -1,
			b:       0,
			wantErr: apperror.ErrDivisionByZero,
		},
		{
			name:    "by negative zero",
			a:       1,
			b:       negZero,
			wantErr: apperror.ErrDivisionByZero,
		},
		{
			name:    "zero by zero is division by zero, not undefined",
			a:       0,
			b:       0,
			wantErr: apperror.ErrDivisionByZero,
		},
		{
			name:    "largest float by zero is division by zero, not overflow",
			a:       math.MaxFloat64,
			b:       0,
			wantErr: apperror.ErrDivisionByZero,
		},
		{
			name:    "division by zero is reported before rounding",
			a:       1,
			b:       0,
			p:       RoundTo(2),
			wantErr: apperror.ErrDivisionByZero,
		},

		// Upper limit: ±1.7976931348623157e308.
		{
			name: "largest float by one",
			a:    math.MaxFloat64,
			b:    1,
			want: math.MaxFloat64,
		},
		{
			name: "largest float by itself is one",
			a:    math.MaxFloat64,
			b:    math.MaxFloat64,
			want: 1,
		},
		{
			name: "largest float by the most negative float is minus one",
			a:    math.MaxFloat64,
			b:    -math.MaxFloat64,
			want: -1,
		},
		{
			name: "one by the smallest normal power of ten stays in range",
			a:    1,
			b:    1e-308,
			want: 1e308,
		},
		{
			name:    "largest float by one half overflows",
			a:       math.MaxFloat64,
			b:       0.5,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "negative overflow",
			a:       -math.MaxFloat64,
			b:       0.5,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "one by the smallest positive float overflows",
			a:       1,
			b:       math.SmallestNonzeroFloat64,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "one by the tiny negative float overflows negatively",
			a:       1,
			b:       -math.SmallestNonzeroFloat64,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "large by tiny overflows",
			a:       1e300,
			b:       1e-10,
			wantErr: apperror.ErrResultOverflow,
		},

		// Lower limit: below ~2.5e-324 a quotient underflows to 0 (accepted).
		{
			name: "subnormal quotient is kept",
			a:    1e-310,
			b:    10,
			want: 1e-311,
		},
		{
			name: "smallest float by 1.5 rounds up to the smallest float",
			a:    math.SmallestNonzeroFloat64,
			b:    1.5,
			want: math.SmallestNonzeroFloat64,
		},
		{
			name: "smallest float halved underflows to zero",
			a:    math.SmallestNonzeroFloat64,
			b:    2,
			want: 0,
		},
		{
			name: "smallest float by largest float underflows to zero",
			a:    math.SmallestNonzeroFloat64,
			b:    math.MaxFloat64,
			want: 0,
		},
		{
			name: "tiny by huge underflows to zero",
			a:    1e-300,
			b:    1e300,
			want: 0,
		},
		{
			name: "negative underflow gives positive zero",
			a:    -1e-300,
			b:    1e300,
			want: 0,
		},
	}

	calc := NewCalculator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calc.Divide(tt.a, tt.b, tt.p)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Divide(%g, %g, %+v) error = %v, want %v", tt.a, tt.b, tt.p, err, tt.wantErr)
			}
			if got != tt.want || math.Signbit(got) != math.Signbit(tt.want) {
				t.Errorf("Divide(%g, %g, %+v) = %g, want %g", tt.a, tt.b, tt.p, got, tt.want)
			}
		})
	}
}

func TestPower(t *testing.T) {
	negZero := math.Copysign(0, -1)

	tests := []struct {
		name    string
		a, b    float64
		p       Precision
		want    float64
		wantErr error
	}{
		{
			name: "integer power",
			a:    2,
			b:    10,
			want: 1024,
		},
		{
			name: "fractional exponent",
			a:    2,
			b:    0.5,
			want: 1.4142135623730951,
		},
		{
			name: "negative exponent",
			a:    10,
			b:    -2,
			want: 0.01,
		},
		{
			name: "negative base to an odd exponent is negative",
			a:    -2,
			b:    3,
			want: -8,
		},
		{
			name: "negative base to an even exponent is positive",
			a:    -2,
			b:    2,
			want: 4,
		},
		{
			name: "negative base to a negative odd exponent",
			a:    -2,
			b:    -3,
			want: -0.125,
		},
		{
			name: "float noise is kept without precision",
			a:    1.1,
			b:    2,
			want: 1.2100000000000002,
		},
		{
			name: "rounds to precision",
			a:    1.1,
			b:    2,
			p:    RoundTo(2),
			want: 1.21,
		},
		{
			name: "rounding to negative zero gives positive zero",
			a:    -0.1,
			b:    3,
			p:    RoundTo(2),
			want: 0,
		},

		// x^0 and x^1.
		{
			name: "any number to the zero is one",
			a:    5,
			b:    0,
			want: 1,
		},
		{
			name: "zero to the zero is one",
			a:    0,
			b:    0,
			want: 1,
		},
		{
			name: "negative zero to the zero is one",
			a:    negZero,
			b:    0,
			want: 1,
		},
		{
			name: "largest float to the zero is one",
			a:    math.MaxFloat64,
			b:    0,
			want: 1,
		},
		{
			name: "one to a huge exponent is one",
			a:    1,
			b:    1e308,
			want: 1,
		},
		{
			name: "minus one to a huge even exponent is one",
			a:    -1,
			b:    1e300,
			want: 1,
		},
		{
			name: "minus one to the largest odd integer float is minus one",
			a:    -1,
			b:    1<<53 - 1,
			want: -1,
		},

		// Zero base.
		{
			name: "zero to a positive power is zero",
			a:    0,
			b:    5,
			want: 0,
		},
		{
			name: "negative zero to an odd power is positive zero",
			a:    negZero,
			b:    3,
			want: 0,
		},
		{
			name:    "zero to a negative power",
			a:       0,
			b:       -1,
			wantErr: apperror.ErrZeroNegativePower,
		},
		{
			name:    "negative zero to a negative odd power",
			a:       negZero,
			b:       -1,
			wantErr: apperror.ErrZeroNegativePower,
		},
		{
			name:    "negative zero to a negative even power",
			a:       negZero,
			b:       -2,
			wantErr: apperror.ErrZeroNegativePower,
		},
		{
			name:    "zero to a negative fractional power",
			a:       0,
			b:       -0.5,
			wantErr: apperror.ErrZeroNegativePower,
		},
		{
			name:    "zero to a negative power is reported before rounding",
			a:       0,
			b:       -1,
			p:       RoundTo(2),
			wantErr: apperror.ErrZeroNegativePower,
		},

		// Negative base, fractional exponent: no real result.
		{
			name:    "square root of a negative base",
			a:       -8,
			b:       0.5,
			wantErr: apperror.ErrResultUndefined,
		},
		{
			name:    "cube root of a negative base is undefined because 1/3 isn't exact",
			a:       -8,
			b:       1.0 / 3,
			wantErr: apperror.ErrResultUndefined,
		},
		{
			name:    "negative base to one and a half",
			a:       -2,
			b:       1.5,
			wantErr: apperror.ErrResultUndefined,
		},
		{
			name:    "undefined is reported before rounding",
			a:       -8,
			b:       0.5,
			p:       RoundTo(2),
			wantErr: apperror.ErrResultUndefined,
		},

		// Integer exponents are exact (math.Pow is not).
		{
			name: "largest power of ten is exact",
			a:    10,
			b:    308,
			want: 1e308,
		},
		{
			name: "smallest normal power of ten is exact",
			a:    10,
			b:    -308,
			want: 1e-308,
		},
		{
			name: "integer power above 2^53 is the nearest float",
			a:    3,
			b:    40,
			want: 12157665459056928801, // the exact power, rounded to float64 by the compiler
		},
		{
			name: "power of the next float above one",
			a:    1.0000000000000002,
			b:    1 << 30,
			want: 1.0000002384186075,
		},
		{
			name: "base near one to a huge integer exponent is e",
			a:    1.0000000000000002,
			b:    1 << 52,
			want: math.E,
		},
		{
			name: "base just below one to a huge integer exponent is 1/e",
			a:    1 - 1.0/(1<<53),
			b:    1 << 53,
			want: 0.3678794411714423,
		},
		{
			name: "base near one to ten billion stays finite and exact",
			a:    1.0000000001,
			b:    1e10,
			want: 2.7182820532347876,
		},
		{
			name: "base just below one to a trillion stays finite and exact",
			a:    0.9999999999,
			b:    1e12,
			want: 3.7200451775013044e-44,
		},
		{
			name:    "NaN base is undefined, not a panic",
			a:       math.NaN(),
			b:       2,
			wantErr: apperror.ErrResultUndefined,
		},

		// Upper limit: ±1.7976931348623157e308.
		{
			name: "largest power of two in range",
			a:    2,
			b:    1023,
			want: 8.98846567431158e307,
		},
		{
			name: "largest float to the first power",
			a:    math.MaxFloat64,
			b:    1,
			want: math.MaxFloat64,
		},
		{
			name:    "first power of two out of range overflows",
			a:       2,
			b:       1024,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "positive overflow",
			a:       10,
			b:       309,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "negative overflow",
			a:       -10,
			b:       309,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "largest float squared overflows",
			a:       math.MaxFloat64,
			b:       2,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "smallest float to minus one overflows",
			a:       math.SmallestNonzeroFloat64,
			b:       -1,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "tiny base to a negative power overflows",
			a:       1e-300,
			b:       -2,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "base near one to a large integer exponent overflows",
			a:       1.0000001,
			b:       1e10,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "base near one to the largest exact-path exponent overflows",
			a:       1.0000000000000002,
			b:       18446744073709549568, // 2^64 - 2048, the largest float64 below 2^64
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "two to an exponent beyond the exact path overflows",
			a:       2,
			b:       1 << 64,
			wantErr: apperror.ErrResultOverflow,
		},

		// Lower limit: below ~2.5e-324 a power underflows to 0 (accepted).
		{
			name: "smallest power of two in range",
			a:    2,
			b:    -1074,
			want: math.SmallestNonzeroFloat64,
		},
		{
			name: "subnormal power of ten",
			a:    10,
			b:    -323,
			want: 1e-323,
		},
		{
			name: "largest float to minus one is subnormal",
			a:    math.MaxFloat64,
			b:    -1,
			want: 5.562684646268003e-309,
		},
		{
			name: "first power of two below range underflows to zero",
			a:    2,
			b:    -1075,
			want: 0,
		},
		{
			name: "power of ten below range underflows to zero",
			a:    10,
			b:    -400,
			want: 0,
		},
		{
			name: "half to a large integer exponent underflows to zero",
			a:    0.5,
			b:    1e10,
			want: 0,
		},
		{
			name: "base just below one to the largest exact-path exponent underflows",
			a:    1 - 1.0/(1<<53),
			b:    18446744073709549568, // 2^64 - 2048
			want: 0,
		},
		{
			name: "half to an exponent beyond the exact path underflows to zero",
			a:    0.5,
			b:    1e20,
			want: 0,
		},
		{
			name: "negative underflow gives positive zero",
			a:    -10,
			b:    -401,
			want: 0,
		},
	}

	calc := NewCalculator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calc.Power(tt.a, tt.b, tt.p)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Power(%g, %g, %+v) error = %v, want %v", tt.a, tt.b, tt.p, err, tt.wantErr)
			}
			if got != tt.want || math.Signbit(got) != math.Signbit(tt.want) {
				t.Errorf("Power(%g, %g, %+v) = %g, want %g", tt.a, tt.b, tt.p, got, tt.want)
			}
		})
	}
}

func TestPercentage(t *testing.T) {
	tests := []struct {
		name    string
		a, b    float64
		p       Precision
		want    float64
		wantErr error
	}{
		{
			name: "whole percent",
			a:    10,
			b:    50,
			want: 5,
		},
		{
			name: "fractional percent",
			a:    12.5,
			b:    80,
			want: 10,
		},
		{
			name: "above one hundred percent",
			a:    150,
			b:    40,
			want: 60,
		},
		{
			name: "negative percent",
			a:    -20,
			b:    50,
			want: -10,
		},
		{
			name: "percent of a negative number",
			a:    20,
			b:    -50,
			want: -10,
		},
		{
			name: "negative percent of a negative number",
			a:    -20,
			b:    -50,
			want: 10,
		},
		{
			name: "hundred percent is the number itself",
			a:    100,
			b:    123.456,
			want: 123.456,
		},
		{
			name: "zero percent of a negative number is positive zero",
			a:    0,
			b:    -5,
			want: 0,
		},

		// Exactness: both float64 orders of operations get these wrong.
		{
			name: "seven percent of a hundred is exactly seven",
			a:    7,
			b:    100,
			want: 7, // a/100*b gives 7.000000000000001
		},
		{
			name: "two-decimal inputs give the expected decimal",
			a:    15,
			b:    19.99,
			want: 2.9985, // (a*b)/100 and a/100*b give 2.9984999999999995
		},
		{
			name: "repeating result",
			a:    33,
			b:    3,
			want: 0.99,
		},
		{
			name: "inexact binary inputs show in the last digit",
			a:    0.1,
			b:    0.3,
			want: 0.00030000000000000003,
		},
		{
			name: "rounding removes the last-digit noise",
			a:    0.1,
			b:    0.3,
			p:    RoundTo(4),
			want: 0.0003,
		},
		{
			name: "rounding to negative zero gives positive zero",
			a:    -1,
			b:    0.1,
			p:    RoundTo(2),
			want: 0,
		},

		// Upper limit: only a result beyond ±1.797e308 overflows.
		{
			name: "large percent whose product alone would overflow",
			a:    1e307,
			b:    50,
			want: 5e306, // (a*b)/100 gives +Inf
		},
		{
			name: "percent of a large number whose product alone would overflow",
			a:    50,
			b:    1e307,
			want: 5e306,
		},
		{
			name: "hundred percent of the largest float",
			a:    100,
			b:    math.MaxFloat64,
			want: math.MaxFloat64,
		},
		{
			name: "minus hundred percent of the largest float",
			a:    -100,
			b:    math.MaxFloat64,
			want: -math.MaxFloat64,
		},
		{
			name:    "two hundred percent of the largest float overflows",
			a:       200,
			b:       math.MaxFloat64,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "negative overflow",
			a:       -200,
			b:       math.MaxFloat64,
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "huge percent of a huge number overflows",
			a:       1e308,
			b:       1e308,
			wantErr: apperror.ErrResultOverflow,
		},

		// Lower limit: below ~2.5e-324 a result underflows to 0 (accepted).
		{
			name: "smallest float percent of a hundred is the smallest float",
			a:    math.SmallestNonzeroFloat64,
			b:    100,
			want: math.SmallestNonzeroFloat64, // a/100*b gives 0
		},
		{
			name: "one percent of a subnormal",
			a:    1,
			b:    5e-322,
			want: math.SmallestNonzeroFloat64,
		},
		{
			name: "half the smallest float underflows to zero",
			a:    math.SmallestNonzeroFloat64,
			b:    50,
			want: 0,
		},
		{
			name: "tiny percent of a tiny number underflows to zero",
			a:    1e-300,
			b:    1e-300,
			want: 0,
		},
		{
			name: "negative underflow gives positive zero",
			a:    -1e-300,
			b:    1e-300,
			want: 0,
		},

		// Non-finite operands can't reach the service through the API; they
		// fall back to float64 math instead of panicking in big.Rat.
		{
			name:    "NaN operand is undefined",
			a:       math.NaN(),
			b:       50,
			wantErr: apperror.ErrResultUndefined,
		},
		{
			name:    "infinite operand overflows",
			a:       10,
			b:       math.Inf(1),
			wantErr: apperror.ErrResultOverflow,
		},
	}

	calc := NewCalculator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calc.Percentage(tt.a, tt.b, tt.p)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Percentage(%g, %g, %+v) error = %v, want %v", tt.a, tt.b, tt.p, err, tt.wantErr)
			}
			if got != tt.want || math.Signbit(got) != math.Signbit(tt.want) {
				t.Errorf("Percentage(%g, %g, %+v) = %g, want %g", tt.a, tt.b, tt.p, got, tt.want)
			}
		})
	}
}

func TestSqrt(t *testing.T) {
	negZero := math.Copysign(0, -1)

	tests := []struct {
		name    string
		x       float64
		p       Precision
		want    float64
		wantErr error
	}{
		{
			name: "perfect square",
			x:    16,
			want: 4,
		},
		{
			name: "irrational root is exactly rounded",
			x:    2,
			want: 1.4142135623730951,
		},
		{
			name: "rounds to precision",
			x:    2,
			p:    RoundTo(3),
			want: 1.414,
		},
		{
			name: "rounds down below half",
			x:    99, // 9.9498743710662
			p:    RoundTo(1),
			want: 9.9,
		},
		{
			name: "rounds a tie half away from zero",
			x:    2.25, // 1.5
			p:    RoundTo(0),
			want: 2,
		},
		{
			name: "decimal",
			x:    1.5,
			want: 1.224744871391589,
		},
		{
			name: "decimal with an exact root",
			x:    2.25,
			want: 1.5,
		},
		{
			name: "decimal below one has a larger root",
			x:    0.01,
			want: 0.1,
		},
		{
			name: "zero",
			x:    0,
			want: 0,
		},
		{
			name: "one",
			x:    1,
			want: 1,
		},
		{
			name: "negative zero is not negative and gives positive zero",
			x:    negZero,
			want: 0,
		},
		{
			name: "large perfect square",
			x:    1e20,
			want: 1e10,
		},
		{
			name: "integer above 2^53 is read inexactly",
			x:    9007199515875288, // 94906267^2 is ...289, which float64 holds as ...288
			want: 94906267,
		},

		// Limits: a square root can't overflow or underflow.
		{
			name: "largest float stays in range",
			x:    math.MaxFloat64,
			want: 1.3407807929942596e154,
		},
		{
			name: "largest power of ten",
			x:    1e308,
			want: 1e154,
		},
		{
			name: "smallest float is in range",
			x:    math.SmallestNonzeroFloat64,
			want: 2.2227587494850775e-162,
		},

		// Negative input: no real root.
		{
			name:    "negative whole number",
			x:       -4,
			wantErr: apperror.ErrNegativeSquareRoot,
		},
		{
			name:    "negative decimal",
			x:       -2.5,
			wantErr: apperror.ErrNegativeSquareRoot,
		},
		{
			name:    "tiny negative subnormal",
			x:       -1e-320,
			wantErr: apperror.ErrNegativeSquareRoot,
		},
		{
			name:    "most negative float",
			x:       -math.MaxFloat64,
			wantErr: apperror.ErrNegativeSquareRoot,
		},
		{
			name:    "negative is reported before rounding",
			x:       -1,
			p:       RoundTo(2),
			wantErr: apperror.ErrNegativeSquareRoot,
		},
		{
			name:    "NaN is undefined",
			x:       math.NaN(),
			wantErr: apperror.ErrResultUndefined,
		},
	}

	calc := NewCalculator()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calc.Sqrt(tt.x, tt.p)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Sqrt(%g, %+v) error = %v, want %v", tt.x, tt.p, err, tt.wantErr)
			}
			if got != tt.want || math.Signbit(got) != math.Signbit(tt.want) {
				t.Errorf("Sqrt(%g, %+v) = %g, want %g", tt.x, tt.p, got, tt.want)
			}
		})
	}
}

// TestPowerOfTenIsExact checks every power of ten float64 can hold against
// the decimal literal strconv parses. math.Pow gets 505 of these 632 wrong,
// which is why integer exponents go through intPow.
func TestPowerOfTenIsExact(t *testing.T) {
	calc := NewCalculator()
	for n := -323; n <= 308; n++ {
		literal := fmt.Sprintf("1e%d", n)
		want, err := strconv.ParseFloat(literal, bitSize)
		if err != nil {
			t.Fatalf("ParseFloat(%q) error = %v", literal, err)
		}
		got, err := calc.Power(10, float64(n), Precision{})
		if err != nil {
			t.Fatalf("Power(10, %d) error = %v", n, err)
		}
		if got != want {
			t.Errorf("Power(10, %d) = %v, want %v", n, got, want)
		}
	}
}

// TestFinish covers the result checks every operation shares, including NaN,
// which no operation so far can produce from finite operands (Divide rejects
// 0/0 as division by zero before dividing).
func TestFinish(t *testing.T) {
	tests := []struct {
		name    string
		r       float64
		p       Precision
		want    float64
		wantErr error
	}{
		{
			name: "finite result passes through",
			r:    12.5,
			want: 12.5,
		},
		{
			name: "negative zero becomes positive zero",
			r:    math.Copysign(0, -1),
			want: 0,
		},
		{
			name:    "NaN is undefined",
			r:       math.NaN(),
			wantErr: apperror.ErrResultUndefined,
		},
		{
			name:    "NaN is undefined even with precision",
			r:       math.NaN(),
			p:       RoundTo(2),
			wantErr: apperror.ErrResultUndefined,
		},
		{
			name:    "positive infinity is overflow",
			r:       math.Inf(1),
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name:    "negative infinity is overflow",
			r:       math.Inf(-1),
			wantErr: apperror.ErrResultOverflow,
		},
		{
			name: "positive tie rounds away from zero",
			r:    2.5,
			p:    RoundTo(0),
			want: 3,
		},
		{
			name: "negative tie rounds away from zero",
			r:    -2.5,
			p:    RoundTo(0),
			want: -3,
		},
		{
			name: "half rounds up, not to even",
			r:    0.5,
			p:    RoundTo(0),
			want: 1,
		},
		{
			name: "rounds a number printed with an exponent",
			r:    1.23456e-7,
			p:    RoundTo(9),
			want: 1.23e-7,
		},
		{
			name: "rounds a tiny number to zero",
			r:    1e-300,
			p:    RoundTo(MaxPrecision),
			want: 0,
		},
		{
			name: "large number has no decimals to round",
			r:    1e300,
			p:    RoundTo(2),
			want: 1e300,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := finish(tt.r, tt.p)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("finish(%g, %+v) error = %v, want %v", tt.r, tt.p, err, tt.wantErr)
			}
			if got != tt.want || math.Signbit(got) != math.Signbit(tt.want) {
				t.Errorf("finish(%g, %+v) = %g, want %g", tt.r, tt.p, got, tt.want)
			}
		})
	}
}
