package service

import (
	"errors"
	"math"
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

// TestFinish covers the result checks every operation shares, including NaN,
// which no operation so far can produce from finite operands.
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
