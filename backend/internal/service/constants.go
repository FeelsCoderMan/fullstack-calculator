package service

// Rounding limits for Precision. float64 holds about 15-17 significant
// digits, so more decimal places than MaxPrecision add only noise.
const (
	MinPrecision = 0
	MaxPrecision = 15
)

// Exact integer powers (intPow).
const (
	// powPrecision is the big.Float mantissa size in bits. It is far beyond
	// float64's 53, so rounding the exact power once gives the nearest float64.
	powPrecision = 2048
	// maxExactExponent bounds |exponent| for intPow: every integer float64
	// below 2^64 fits in a uint64 and takes at most 63 squarings (~90µs). At
	// and beyond it, every base other than 0 and ±1 overflows or underflows
	// ((1+2^-52)^(2^64) is about e^4096), which math.Pow gets right.
	maxExactExponent = 1 << 64
)

// float64 formatting used by round.
const (
	// shortestFormat and shortestPrecision print the shortest decimal that
	// parses back to the same float64 (2.675, not 2.67499999...).
	shortestFormat    = 'g'
	shortestPrecision = -1
	bitSize           = 64
)
