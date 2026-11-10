package service

// Rounding limits for Precision. float64 holds about 15-17 significant
// digits, so more decimal places than MaxPrecision add only noise.
const (
	MinPrecision = 0
	MaxPrecision = 15
)

// float64 formatting used by round.
const (
	// shortestFormat and shortestPrecision print the shortest decimal that
	// parses back to the same float64 (2.675, not 2.67499999...).
	shortestFormat    = 'g'
	shortestPrecision = -1
	bitSize           = 64
)
